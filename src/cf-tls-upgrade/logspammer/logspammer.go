package logspammer

import (
	"acceptance-tests/testing/helpers"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/cloudfoundry/sonde-go/events"
)

type Spammer struct {
	sync.Mutex
	appURL      string
	frequency   time.Duration
	doneGet     chan struct{}
	doneMsg     chan struct{}
	wg          sync.WaitGroup
	logMessages []string
	logWritten  int
	msgChan     <-chan *events.Envelope
	errors      helpers.ErrorSet
	prefix      string
}

func NewSpammer(appURL string, msgChan <-chan *events.Envelope, frequency time.Duration) *Spammer {
	return &Spammer{
		appURL:      appURL,
		frequency:   frequency,
		doneGet:     make(chan struct{}),
		doneMsg:     make(chan struct{}),
		msgChan:     msgChan,
		errors:      helpers.ErrorSet{},
		prefix:      fmt.Sprintf("spammer-%d", rand.Int()),
		logMessages: []string{},
	}
}

func (s *Spammer) CheckStream() bool {
	resp, err := http.Get(fmt.Sprintf("%s/log/TEST", s.appURL))
	if err != nil {
		return false
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	err = resp.Body.Close()
	if err != nil {
		return false
	}

	select {
	case <-s.msgChan:
		return true
	case <-time.After(5 * time.Millisecond):
		return false
	}
}

func (s *Spammer) Start() error {
	go func() {
		s.wg.Add(1)
		for {
			select {
			case <-s.doneGet:
				s.wg.Done()
				return
			case <-time.After(s.frequency):
				resp, err := http.Get(fmt.Sprintf("%s/log/%s-%d-", s.appURL, s.prefix, s.logWritten))
				if err != nil {
					s.errors.Add(err)
					continue
				}

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					s.errors.Add(err)
					continue
				}

				err = resp.Body.Close()
				if err != nil {
					s.errors.Add(err)
					continue
				}

				if resp.StatusCode != http.StatusOK {
					s.errors.Add(fmt.Errorf("%+v -- %v", resp, string(body)))
					continue
				}

				s.logWritten++
			}
		}
	}()

	go func() {
		for {
			select {
			case <-s.doneMsg:
				return
			case msg := <-s.msgChan:
				s.Lock()
				if msg != nil {
					if msg.LogMessage != nil {
						if *msg.LogMessage.SourceType == "APP" && *msg.LogMessage.MessageType == events.LogMessage_OUT {
							s.logMessages = append(s.logMessages, string(msg.LogMessage.Message))
						}
					}
				}
				s.Unlock()
			}
		}
	}()

	return nil
}

func (s *Spammer) Stop() error {
	close(s.doneGet)
	s.wg.Wait()
	time.Sleep(1 * time.Second)
	close(s.doneMsg)
	return nil
}

func (s *Spammer) Check() error {
	diff := s.logWritten - len(s.LogMessages())
	if diff > 0 {
		s.errors["missing log lines"] = diff
	}

	if len(s.errors) > 0 {
		return s.errors
	}

	return nil
}

func (s *Spammer) LogMessages() []string {
	s.Lock()
	defer s.Unlock()

	return s.logMessages
}
