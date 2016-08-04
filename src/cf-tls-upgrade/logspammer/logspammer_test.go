package logspammer_test

import (
	"cf-tls-upgrade/logspammer"

	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudfoundry/sonde-go/events"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeNoaaConsumer struct {
	StreamCall struct {
		CallCount int
		Receives  struct {
			AppGuid   string
			AuthToken string
		}
		Returns struct {
			OutputChan chan *events.Envelope
			ErrChan    chan error
		}
	}
}

func (f *fakeNoaaConsumer) Stream(appGuid string, authToken string) (outputChan <-chan *events.Envelope, errorChan <-chan error) {
	f.StreamCall.CallCount++
	f.StreamCall.Receives.AppGuid = appGuid
	f.StreamCall.Receives.AuthToken = authToken

	return f.StreamCall.Returns.OutputChan, f.StreamCall.Returns.ErrChan
}

var _ = Describe("logspammer", func() {
	var (
		appServer          *httptest.Server
		spammer            *logspammer.Spammer
		noaaConsumer       *fakeNoaaConsumer
		appServerCallCount int32
		skipStream         bool
		l                  sync.Mutex
		logChannelLock     sync.Mutex
	)

	var setSkipStream = func(b bool) {
		l.Lock()
		defer l.Unlock()
		skipStream = b
	}

	var getSkipStream = func() bool {
		l.Lock()
		defer l.Unlock()
		return skipStream
	}

	var writeLogToChannel = func(envelope *events.Envelope) {
		logChannelLock.Lock()
		defer logChannelLock.Unlock()

		noaaConsumer.StreamCall.Returns.OutputChan <- envelope
	}

	BeforeEach(func() {
		atomic.StoreInt32(&appServerCallCount, 0)
		noaaConsumer = &fakeNoaaConsumer{}
		noaaConsumer.StreamCall.Returns.OutputChan = make(chan *events.Envelope)

		appServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/log") && req.Method == "GET" {
				parts := strings.Split(req.URL.Path, "/")
				w.WriteHeader(http.StatusOK)
				if getSkipStream() == false {
					sourceType := "APP"
					messageType := events.LogMessage_OUT

					envelope := &events.Envelope{
						LogMessage: &events.LogMessage{
							MessageType: &messageType,
							SourceType:  &sourceType,
							Message:     []byte(parts[2]),
						},
					}
					writeLogToChannel(envelope)
				}
				atomic.AddInt32(&appServerCallCount, 1)
				return
			}
			w.WriteHeader(http.StatusTeapot)
		}))

		spammer = logspammer.NewSpammer(appServer.URL, noaaConsumer.StreamCall.Returns.OutputChan, 10*time.Millisecond)
		err := spammer.Start()
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() int {
			return len(spammer.LogMessages())
		}).Should(BeNumerically(">", 0))
	})

	Describe("Check", func() {
		It("check for log interruptions", func() {
			err := spammer.Stop()
			Expect(err).NotTo(HaveOccurred())

			err = spammer.Check()
			Expect(err).NotTo(HaveOccurred())

			Expect(int(atomic.LoadInt32(&appServerCallCount))).To(Equal(len(spammer.LogMessages())))
		})

		Context("failure cases", func() {
			It("returns an error when the spammer fails to write to the app", func() {
				spammer.Stop()
				spammer = logspammer.NewSpammer("", make(chan *events.Envelope), 0)
				err := spammer.Start()
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(100 * time.Millisecond)

				err = spammer.Stop()
				Expect(err).NotTo(HaveOccurred())

				err = spammer.Check()
				Expect(err).To(MatchError(ContainSubstring("unsupported protocol scheme")))
			})

			It("returns an error when an app log line is missing", func() {
				setSkipStream(true)
				logCount := len(spammer.LogMessages())

				Eventually(func() int {
					return int(atomic.LoadInt32(&appServerCallCount))
				}, "1m", "10ms").Should(BeNumerically(">", logCount+10))

				setSkipStream(false)

				err := spammer.Stop()
				Expect(err).NotTo(HaveOccurred())

				logdiff := int(atomic.LoadInt32(&appServerCallCount)) - len(spammer.LogMessages())

				err = spammer.Check()
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("missing log lines : %v", logdiff))))
			})
		})
	})

	Describe("Start", func() {
		It("ignores non source_type APP messages", func() {
			sourceType := "not-app-source-type"
			messageType := events.LogMessage_OUT

			envelope := &events.Envelope{
				LogMessage: &events.LogMessage{
					MessageType: &messageType,
					SourceType:  &sourceType,
					Message:     []byte("NOT-AN-APP-MESSAGE"),
				},
			}
			writeLogToChannel(envelope)

			time.Sleep(5 * time.Millisecond)
			err := spammer.Stop()
			Expect(err).NotTo(HaveOccurred())

			messages := spammer.LogMessages()

			Expect(messages).ToNot(HaveLen(0))

			for _, message := range messages {
				Expect(message).ToNot(ContainSubstring("NOT-AN-APP-MESSAGE"))
			}

		})

		It("ignores nil log message", func() {
			sourceType := "APP"
			messageType := events.LogMessage_OUT
			envelopeNilMessage := &events.Envelope{
				LogMessage: nil,
			}

			envelope := &events.Envelope{
				LogMessage: &events.LogMessage{
					MessageType: &messageType,
					SourceType:  &sourceType,
					Message:     []byte("Message written after nil message"),
				},
			}

			writeLogToChannel(envelopeNilMessage)
			writeLogToChannel(envelope)

			Eventually(func() bool {
				messages := spammer.LogMessages()
				for _, message := range messages {
					if strings.Contains(message, "Message written after nil message") {
						return true
					}
				}

				return false
			}).Should(BeTrue())

			err := spammer.Stop()
			Expect(err).NotTo(HaveOccurred())
		})

		It("ignores non message_type OUT messages", func() {
			sourceType := "APP"
			messageType := events.LogMessage_ERR

			envelope := &events.Envelope{
				LogMessage: &events.LogMessage{
					MessageType: &messageType,
					SourceType:  &sourceType,
					Message:     []byte("NOT-AN-OUT-MESSAGE"),
				},
			}
			writeLogToChannel(envelope)

			time.Sleep(5 * time.Millisecond)
			err := spammer.Stop()
			Expect(err).NotTo(HaveOccurred())

			messages := spammer.LogMessages()

			Expect(messages).ToNot(HaveLen(0))

			for _, message := range messages {
				Expect(message).ToNot(ContainSubstring("NOT-AN-OUT-MESSAGE"))
			}
		})
	})

	Describe("Stop", func() {
		It("no longer streams messages when the spammer has been stopped", func() {
			err := spammer.Stop()
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				select {
				case noaaConsumer.StreamCall.Returns.OutputChan <- nil:
					return true
				case <-time.After(10 * time.Millisecond):
					return false
				}
			}, "100ms", "10ms").Should(BeFalse())
		})
	})
})
