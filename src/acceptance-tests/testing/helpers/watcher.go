package helpers

import (
	"sync"

	goetcd "github.com/coreos/go-etcd/etcd"
)

type etcdWatcher interface {
	Watch(prefix string, waitIndex uint64, recursive bool,
		receiver chan *goetcd.Response, stop chan bool) (*goetcd.Response, error)
}

type Watcher struct {
	Response  chan *goetcd.Response
	Stop      chan bool
	stopMutex sync.Mutex
	dataMutex sync.Mutex
	data      map[string]string
	stopped   bool
	err       error
}

func Watch(watcher etcdWatcher, prefix string) *Watcher {
	w := &Watcher{
		data:     map[string]string{},
		Response: make(chan *goetcd.Response),
		Stop:     make(chan bool),
	}
	go func() {
		for {
			r, ok := <-w.Response
			if !ok {
				return
			}
			if r != nil && r.Node != nil {
				w.AddData(r.Node.Key, r.Node.Value)
			}
		}
	}()

	go func() {
		_, err := watcher.Watch(prefix, 0, true, w.Response, w.Stop)
		w.setStoppedAndError(true, err)
	}()

	return w
}

func (w *Watcher) setStoppedAndError(stopped bool, err error) {
	w.stopMutex.Lock()
	defer w.stopMutex.Unlock()

	w.stopped = stopped
	w.err = err
}

func (w *Watcher) IsStopped() bool {
	w.stopMutex.Lock()
	defer w.stopMutex.Unlock()

	return w.stopped
}

func (w *Watcher) GetEtcdWatchError() error {
	w.stopMutex.Lock()
	defer w.stopMutex.Unlock()

	return w.err
}

func (w *Watcher) AddData(key, value string) {
	w.dataMutex.Lock()
	defer w.dataMutex.Unlock()

	w.data[key] = value
}

func (w *Watcher) Data() map[string]string {
	w.dataMutex.Lock()
	defer w.dataMutex.Unlock()

	return w.data
}
