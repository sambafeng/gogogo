package integration_test

import (
	"github.com/gogo/protobuf/proto"
	"fmt"
	"github.com/cloudfoundry/dropsonde/autowire"
	"github.com/cloudfoundry/dropsonde/autowire/metrics"
	"github.com/cloudfoundry/dropsonde/control"
	"github.com/cloudfoundry/dropsonde/events"
	"github.com/cloudfoundry/dropsonde/factories"
	"github.com/cloudfoundry/dropsonde/metric_sender"
	uuid "github.com/nu7hatch/gouuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// these tests need to be invoked individually from an external script,
// since environment variables need to be set/unset before starting the tests
var _ = Describe("Autowire End-to-End", func() {
	Context("with DROPSONDE_ORIGIN set", func() {
		var oldEnv string

		BeforeEach(func() {
			oldEnv = os.Getenv("DROPSONDE_ORIGIN")
			os.Setenv("DROPSONDE_ORIGIN", "test-origin")
			emitter, _ := autowire.CreateDefaultEmitter()
			autowire.Initialize(emitter)
			metrics.Initialize(metric_sender.NewMetricSender(autowire.AutowiredEmitter()))
		})

		AfterEach(func() {
			os.Setenv("DROPSONDE_ORIGIN", oldEnv)
		})

		It("emits HTTP client/server events and heartbeats", func() {
			udpListener, err := net.ListenPacket("udp4", ":3457")
			Expect(err).ToNot(HaveOccurred())
			defer udpListener.Close()
			udpDataChan := make(chan []byte, 16)

			receivedEvents := make(map[string]bool)
			heartbeatUuidsChan := make(chan string, 1000)

			lock := sync.RWMutex{}
			origin := os.Getenv("DROPSONDE_ORIGIN")
			heartbeatRequest := newHeartbeatRequest()
			marshalledHeartbeatRequest, _ := proto.Marshal(heartbeatRequest)

			requestedHeartbeat := false
			go func() {
				defer close(udpDataChan)
				for {
					buffer := make([]byte, 1024)
					n, addr, err := udpListener.ReadFrom(buffer)
					if err != nil {
						return
					}

					if !requestedHeartbeat {

						udpListener.WriteTo(marshalledHeartbeatRequest, addr)
						requestedHeartbeat = true
					}

					if n == 0 {
						panic("Received empty packet")
					}
					envelope := new(events.Envelope)
					err = proto.Unmarshal(buffer[0:n], envelope)
					if err != nil {
						panic(err)
					}

					var eventId = envelope.GetEventType().String()

					switch envelope.GetEventType() {
					case events.Envelope_HttpStart:
						eventId += envelope.GetHttpStart().GetPeerType().String()
					case events.Envelope_HttpStop:
						eventId += envelope.GetHttpStop().GetPeerType().String()
					case events.Envelope_Heartbeat:
						heartbeatUuidsChan <- envelope.GetHeartbeat().GetControlMessageIdentifier().String()
					case events.Envelope_ValueMetric:
						eventId += envelope.GetValueMetric().GetName()
					case events.Envelope_CounterEvent:
						eventId += envelope.GetCounterEvent().GetName()
					default:
						panic("Unexpected message type")

					}

					if envelope.GetOrigin() != origin {
						panic("origin not as expected")
					}

					func() {
						lock.Lock()
						defer lock.Unlock()
						receivedEvents[eventId] = true
					}()
				}
			}()

			httpListener, err := net.Listen("tcp", "localhost:0")
			Expect(err).ToNot(HaveOccurred())
			defer httpListener.Close()
			httpHandler := autowire.InstrumentedHandler(FakeHandler{})
			go http.Serve(httpListener, httpHandler)

			_, err = http.Get("http://" + httpListener.Addr().String())
			Expect(err).ToNot(HaveOccurred())

			metrics.SendValue("TestMetric", 0, "")
			metrics.IncrementCounter("TestIncrementCounter")

			expectedEventTypes := []string{"HttpStartClient", "HttpStartServer", "HttpStopServer", "HttpStopClient", "ValueMetricnumCPUS", "ValueMetricTestMetric", "CounterEventTestIncrementCounter"}

			for _, eventType := range expectedEventTypes {
				Eventually(func() bool {
					lock.RLock()
					defer lock.RUnlock()
					_, ok := receivedEvents[eventType]
					return ok
				}).Should(BeTrue(), fmt.Sprintf("missing %s", eventType))
			}

			heartbeatUuid := heartbeatRequest.GetIdentifier().String()
			Eventually(heartbeatUuidsChan).Should(Receive(Equal(heartbeatUuid)))

		})
	})
})

type FakeHandler struct{}

func (fh FakeHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(rw, "Hello")
}

type FakeRoundTripper struct{}

func (frt FakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, nil
}

func newHeartbeatRequest() *control.ControlMessage {
	id, _ := uuid.NewV4()

	return &control.ControlMessage{
		Origin:      proto.String("MET"),
		Identifier:  factories.NewControlUUID(id),
		Timestamp:   proto.Int64(time.Now().UnixNano()),
		ControlType: control.ControlMessage_HeartbeatRequest.Enum(),
	}
}
