package helpers

import goetcd "github.com/coreos/go-etcd/etcd"

type etcdWatcher interface {
	Watch(prefix string, waitIndex uint64, recursive bool,
		receiver chan *goetcd.Response, stop chan bool) (*goetcd.Response, error)
}

type Watcher struct {
	Data     map[string]string
	Response chan *goetcd.Response
	Stop     chan bool
	Stopped  bool
	Error    error
}

func Watch(watcher etcdWatcher, prefix string) *Watcher {
	w := &Watcher{
		Data:     map[string]string{},
		Response: make(chan *goetcd.Response),
		Stop:     make(chan bool),
	}

	go func() {
		_, err := watcher.Watch(prefix, 0, true, w.Response, w.Stop)
		w.Stopped = true
		w.Error = err
	}()

	go func() {
		for {
			r, ok := <-w.Response
			if !ok {
				return
			}
			if r != nil && r.Node != nil {
				w.Data[r.Node.Key] = r.Node.Value
			}
		}
	}()

	return w
}
