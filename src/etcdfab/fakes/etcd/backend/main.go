package backend

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
)

type etcdBackend struct {
	callCount  int32
	args       []string
	fastFail   bool
	shouldExit bool

	mutex sync.Mutex
}

type EtcdBackendServer struct {
	server  *httptest.Server
	backend *etcdBackend
}

func NewEtcdBackendServer() *EtcdBackendServer {
	etcdBackendServer := &EtcdBackendServer{
		backend: &etcdBackend{},
	}
	etcdBackendServer.server = httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		etcdBackendServer.ServeHTTP(responseWriter, request)
	}))

	return etcdBackendServer
}

func (e *EtcdBackendServer) Reset() {
	e.backend = &etcdBackend{}
}

func (e *EtcdBackendServer) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/call":
		e.call(responseWriter, request)
	case "/exit":
		e.exit(responseWriter, request)
	}
}

func (e *EtcdBackendServer) exit(responseWriter http.ResponseWriter, request *http.Request) {
	if e.backend.shouldExit {
		responseWriter.WriteHeader(http.StatusOK)
		e.backend.shouldExit = true
	} else {
		responseWriter.WriteHeader(http.StatusTeapot)
	}
}

func (e *EtcdBackendServer) call(responseWriter http.ResponseWriter, request *http.Request) {
	atomic.AddInt32(&e.backend.callCount, 1)
	argsJSON, err := ioutil.ReadAll(request.Body)
	if err != nil {
		panic(err)
	}

	var args []string
	err = json.Unmarshal(argsJSON, &args)
	if err != nil {
		panic(err)
	}

	e.setArgs(args)

	if e.FastFail() {
		responseWriter.WriteHeader(http.StatusInternalServerError)
	} else {
		responseWriter.WriteHeader(http.StatusOK)
	}
}

func (e *EtcdBackendServer) setArgs(args []string) {
	e.backend.mutex.Lock()
	defer e.backend.mutex.Unlock()
	e.backend.args = args
}

func (e *EtcdBackendServer) EnableFastFail() {
	e.backend.mutex.Lock()
	defer e.backend.mutex.Unlock()
	e.backend.fastFail = true
}

func (e *EtcdBackendServer) DisableFastFail() {
	e.backend.mutex.Lock()
	defer e.backend.mutex.Unlock()
	e.backend.fastFail = false
}

func (e *EtcdBackendServer) FastFail() bool {
	e.backend.mutex.Lock()
	defer e.backend.mutex.Unlock()
	return e.backend.fastFail
}

func (e *EtcdBackendServer) Exit() {
	e.backend.mutex.Lock()
	defer e.backend.mutex.Unlock()
	e.backend.shouldExit = true
}

func (e *EtcdBackendServer) ServerURL() string {
	return e.server.URL
}

func (e *EtcdBackendServer) GetCallCount() int {
	return int(atomic.LoadInt32(&e.backend.callCount))
}

func (e *EtcdBackendServer) GetArgs() []string {
	e.backend.mutex.Lock()
	defer e.backend.mutex.Unlock()
	return e.backend.args
}
