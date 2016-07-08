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
		watcher = helpers.Watch(fakeWatcher, "/")
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

	Context("failure cases", func() {
		It("assigns an error", func() {
			fakeWatcher.WatchCall.Returns.Error = errors.New("something bad happened")
			watcher.Stop <- true

			Eventually(watcher.IsStopped, "10s", "1s").Should(BeTrue())
			Expect(watcher.GetEtcdWatchError()).To(MatchError("something bad happened"))
		})
	})
})
