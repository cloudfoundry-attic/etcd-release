package helpers_test

import (
	"acceptance-tests/testing/helpers"
	"errors"
	"fmt"

	goetcd "github.com/coreos/go-etcd/etcd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeEtcdWatcher struct {
	WatchCall struct {
		Started  chan bool
		Receives struct {
			Prefix    string
			WaitIndex uint64
			Recursive bool
			Receiver  chan *goetcd.Response
			Stop      chan bool
		}
		Returns struct {
			Response *goetcd.Response
			Error    error
		}
	}
}

func (w *fakeEtcdWatcher) Watch(prefix string, waitIndex uint64, recursive bool,
	receiver chan *goetcd.Response, stop chan bool) (*goetcd.Response, error) {
	w.WatchCall.Receives.Prefix = prefix
	w.WatchCall.Receives.WaitIndex = waitIndex
	w.WatchCall.Receives.Recursive = recursive
	w.WatchCall.Receives.Receiver = receiver
	w.WatchCall.Receives.Stop = stop

	defer close(w.WatchCall.Receives.Receiver)

	w.WatchCall.Started <- true
	<-w.WatchCall.Receives.Stop

	return w.WatchCall.Returns.Response, w.WatchCall.Returns.Error
}

var _ = Describe("Watcher", func() {
	var (
		fakeWatcher *fakeEtcdWatcher
		watcher     *helpers.Watcher
	)

	BeforeEach(func() {
		fakeWatcher = &fakeEtcdWatcher{}
		fakeWatcher.WatchCall.Started = make(chan bool, 2)
		watcher = helpers.Watch(fakeWatcher, "/")
		<-fakeWatcher.WatchCall.Started
	})

	It("watches and records key changes", func() {
		go func() {
			for i := 1; i <= 4; i++ {
				watcher.Response <- &goetcd.Response{
					Node: &goetcd.Node{
						Value: fmt.Sprintf("value%d", i),
						Key:   fmt.Sprintf("key%d", i),
					},
				}
			}
			watcher.Stop <- true
		}()

		Eventually(watcher.IsStopped, "10s", "1s").Should(BeTrue())
		Expect(watcher.Data()).To(Equal(map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
			"key4": "value4",
		}))
	})

	It("starts the watcher up if it is prematurely closed", func() {
		watcher.Response <- &goetcd.Response{
			Node: &goetcd.Node{
				Value:         "value1",
				Key:           "key1",
				ModifiedIndex: 1,
			},
		}
		fakeWatcher.WatchCall.Returns.Error = errors.New("EOF")
		fakeWatcher.WatchCall.Receives.Stop <- true
		<-fakeWatcher.WatchCall.Started
		watcher.Response <- &goetcd.Response{
			Node: &goetcd.Node{
				Value:         "value2",
				Key:           "key2",
				ModifiedIndex: 2,
			},
		}
		watcher.Response <- &goetcd.Response{
			Node: &goetcd.Node{
				Value:         "value3",
				Key:           "key3",
				ModifiedIndex: 3,
			},
		}
		fakeWatcher.WatchCall.Returns.Error = nil
		watcher.Stop <- true

		Eventually(watcher.IsStopped, "10s", "1s").Should(BeTrue())
		Expect(fakeWatcher.WatchCall.Receives.WaitIndex).To(Equal(uint64(2)))
		Expect(watcher.Data()).To(Equal(map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}))
	})

	It("does not panic when the watcher has been closed", func() {
		watcher.Stop <- true

		Eventually(watcher.IsStopped, "2s", "1s").Should(BeTrue())
		Expect(watcher.GetEtcdWatchError()).To(BeNil())
	})

	It("does not panic when the response is nil", func() {
		watcher.Response <- nil
		Expect(watcher.IsStopped()).To(BeFalse())
		Expect(watcher.Data()).To(Equal(map[string]string{}))
	})

	It("does not panic when the response node is nil", func() {
		watcher.Response <- &goetcd.Response{Node: nil}
		Expect(watcher.IsStopped()).To(BeFalse())
		Expect(watcher.Data()).To(Equal(map[string]string{}))
	})
})
