package router

import (
	"github.com/cloudfoundry/dropsonde/autowire"
	steno "github.com/cloudfoundry/gosteno"
	vcap "github.com/dinp/gorouter/common"
	"github.com/dinp/gorouter/config"
	"github.com/dinp/gorouter/proxy"
	"github.com/dinp/gorouter/registry"
	"github.com/dinp/gorouter/varz"

	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

var DrainTimeout = errors.New("router: Drain timeout")

type Router struct {
	config    *config.Config
	proxy     proxy.Proxy
	registry  *registry.RouteRegistry
	varz      varz.Varz
	component *vcap.VcapComponent

	listener net.Listener

	logger *steno.Logger
}

func NewRouter(cfg *config.Config, p proxy.Proxy, r *registry.RouteRegistry, v varz.Varz,
	logCounter *vcap.LogCounter) (*Router, error) {

	var host string
	if cfg.Status.Port != 0 {
		host = fmt.Sprintf("%s:%d", cfg.Ip, cfg.Status.Port)
	}

	varz := &vcap.Varz{
		UniqueVarz: v,
		GenericVarz: vcap.GenericVarz{
			LogCounts: logCounter,
		},
	}

	healthz := &vcap.Healthz{}

	component := &vcap.VcapComponent{
		Type:        "Router",
		Index:       cfg.Index,
		Host:        host,
		Credentials: []string{cfg.Status.User, cfg.Status.Pass},
		Config:      cfg,
		Varz:        varz,
		Healthz:     healthz,
		InfoRoutes: map[string]json.Marshaler{
			"/routes": r,
		},
	}

	router := &Router{
		config:    cfg,
		proxy:     p,
		registry:  r,
		varz:      v,
		component: component,
		logger:    steno.NewLogger("router"),
	}

	if err := router.component.Start(); err != nil {
		return nil, err
	}

	return router, nil
}

func (r *Router) Run() <-chan error {
	r.registry.Register()
	r.registry.StartReloadingCycle()

	server := http.Server{
		Handler: autowire.InstrumentedHandler(r.proxy),
	}

	errChan := make(chan error, 1)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", r.config.Port))
	if err != nil {
		r.logger.Fatalf("net.Listen: %s", err)
		errChan <- err
		return errChan
	}

	r.listener = listener
	r.logger.Infof("Listening on %s", listener.Addr())

	go func() {
		err := server.Serve(listener)
		errChan <- err
	}()

	return errChan
}

func (r *Router) Drain(drainTimeout time.Duration) error {
	r.listener.Close()

	drained := make(chan struct{})
	go func() {
		r.proxy.Wait()
		close(drained)
	}()

	select {
	case <-drained:
	case <-time.After(drainTimeout):
		r.logger.Warn("router.drain.timed-out")
		return DrainTimeout
	}
	return nil
}

func (r *Router) Stop() {
	r.listener.Close()
	r.component.Stop()
}
