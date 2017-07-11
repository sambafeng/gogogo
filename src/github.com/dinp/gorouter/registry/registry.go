package registry

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	steno "github.com/cloudfoundry/gosteno"
	"github.com/garyburd/redigo/redis"

	"github.com/dinp/gorouter/config"
	"github.com/dinp/gorouter/route"
)

var byUriTmp map[route.Uri]*route.Pool
var RedisConnPool *redis.Pool

func InitRedisConnPool(c *config.Config) {
	RedisConnPool = &redis.Pool{
		MaxIdle:     100,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", c.RedisServer)
			if err != nil {
				return nil, err
			}
			return conn, err
		},
	}
}

type RouteRegistry struct {
	sync.RWMutex

	logger *steno.Logger

	byUri map[route.Uri]*route.Pool

	reloadUriInterval time.Duration

	ticker           *time.Ticker
	timeOfLastUpdate time.Time
}

func NewRouteRegistry(c *config.Config) *RouteRegistry {
	r := &RouteRegistry{}

	r.logger = steno.NewLogger("router.registry")

	r.byUri = make(map[route.Uri]*route.Pool)

	r.reloadUriInterval = c.ReloadUriInterval

	return r
}

func (r *RouteRegistry) Register() {
	t := time.Now()
	r.Lock()

	r.logger.Debug("registry start to Register")

	byUriTmp, ok := r.GenerateUriMap()
	if ok {
		r.byUri = byUriTmp

		r.timeOfLastUpdate = t
		r.logger.Debug("registry finished Register")
	} else {
		r.logger.Error("generate UriMap failed.")
	}

	r.Unlock()
}

func (r *RouteRegistry) Lookup(uri route.Uri) *route.Pool {
	r.RLock()

	uri = uri.ToLower()
	pool := r.byUri[uri]

	r.RUnlock()

	return pool
}

func (r *RouteRegistry) StartReloadingCycle() {
	go r.Reloading()
}

func (registry *RouteRegistry) NumUris() int {
	registry.RLock()
	uriCount := len(registry.byUri)
	registry.RUnlock()

	return uriCount
}

func (r *RouteRegistry) TimeOfLastUpdate() time.Time {
	r.RLock()
	t := r.timeOfLastUpdate
	r.RUnlock()

	return t
}

func (r *RouteRegistry) NumEndpoints() int {
	r.RLock()
	uris := make(map[string]struct{})
	f := func(endpoint *route.Endpoint) {
		uris[endpoint.CanonicalAddr()] = struct{}{}
	}
	for _, pool := range r.byUri {
		pool.Each(f)
	}
	r.RUnlock()

	return len(uris)
}

func (r *RouteRegistry) MarshalJSON() ([]byte, error) {
	r.RLock()
	defer r.RUnlock()

	return json.Marshal(r.byUri)
}

func (r *RouteRegistry) Reloading() {
	if r.reloadUriInterval == 0 {
		return
	}

	tick := time.Tick(r.reloadUriInterval)
	for {
		select {
		case <-tick:
			r.ReloadUri()
		}
	}
}

func (r *RouteRegistry) ReloadUri() {
	r.Lock()
	defer r.Unlock()

	r.logger.Debug("registry start to ReloadUri")

	byUriTmp, ok := r.GenerateUriMap()
	if ok {
		r.byUri = byUriTmp

		r.timeOfLastUpdate = time.Now()
		r.logger.Debug("registry finished ReloadUri")
	} else {
		r.logger.Error("generate UriMap failed.")
	}
}

func (r *RouteRegistry) GenerateUriMap() (map[route.Uri]*route.Pool, bool) {
	byUriTmp = make(map[route.Uri]*route.Pool)

	rc := RedisConnPool.Get()
	defer rc.Close()

	uriList, err := redis.Strings(rc.Do("KEYS", "*"))
	if err != nil {
		r.logger.Error(err.Error())
		return nil, false
	}

	for _, uriString := range uriList {
		uriType := route.Uri(strings.Split(uriString, "/")[1])
		if uriType == "rs" {
			uri := route.Uri(strings.Split(uriString, "/")[2])
			uri = uri.ToLower()

			pool, found := byUriTmp[uri]
			if !found {
				pool = route.NewPool(r.reloadUriInterval / 5)
				byUriTmp[uri] = pool
			}

			addresslist, err := redis.Strings(rc.Do("LRANGE", uriString, "0", "-1"))
			if err != nil {
				r.logger.Error(err.Error())
				return nil, false
			}

			for _, address := range addresslist {
				host := strings.Split(address, ":")[0]
				p, _ := strconv.Atoi(strings.Split(address, ":")[1])
				port := uint16(p)

				endpoint := route.NewEndpoint(host, port, nil)
				pool.Put(endpoint)
			}
		}
	}

	for _, uriString := range uriList {
		uriType := route.Uri(strings.Split(uriString, "/")[1])
		if uriType == "cname" {
			cname := route.Uri(strings.Split(uriString, "/")[2])
			cname = cname.ToLower()

			_, found := byUriTmp[cname]
			if !found {
				u, err := redis.String(rc.Do("GET", uriString))
				if err != nil {
					r.logger.Error(err.Error())
					return nil, false
				}

				uri := route.Uri(strings.Split(u, "/")[2])
				uri = uri.ToLower()

				pool, found := byUriTmp[uri]
				if !found {
					r.logger.Warnf("cname %s points to uri %s, but uri %s do not have rs", cname, uri, uri)
				} else {
					byUriTmp[cname] = pool
				}
			} else {
				r.logger.Warnf("cname %s has rs", cname)
			}
		}
	}

	return byUriTmp, true
}
