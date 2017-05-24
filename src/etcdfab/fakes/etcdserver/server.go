package etcdserver

import (
	"log"
	"net"
	"net/http"
	"sync"
)

type EtcdServer struct {
	server       *http.Server
	httpListener net.Listener

	backend *etcdBackend

	backendMutex sync.Mutex
}

type etcdBackend struct {
	membersJSON          string
	membersStatusCode    int
	addMembersJSON       string
	addMembersStatusCode int
}

func NewEtcdServer(httpAddr string) *EtcdServer {
	etcdServer := &EtcdServer{
		backend: &etcdBackend{},
	}
	etcdServer.Reset()

	handler := http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		etcdServer.ServeHTTP(responseWriter, request)
	})

	etcdServer.server = &http.Server{
		Addr:    httpAddr,
		Handler: handler,
	}

	var err error
	etcdServer.httpListener, err = net.Listen("tcp", httpAddr)
	if err != nil {
		panic(err)
	}

	go etcdServer.server.Serve(etcdServer.httpListener)

	return etcdServer
}

func (e *EtcdServer) Exit() error {
	err := e.httpListener.Close()
	if err != nil {
		log.Fatalf("Failed to close server: %s\n", err)
	}

	return nil
}

func (e *EtcdServer) Reset() {
	e.backend = &etcdBackend{
		membersStatusCode:    http.StatusOK,
		addMembersStatusCode: http.StatusCreated,
	}
}

func (e *EtcdServer) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/v2/members":
		e.handleMembers(responseWriter, request)
	}
}

func (e *EtcdServer) handleMembers(responseWriter http.ResponseWriter, request *http.Request) {
	e.backendMutex.Lock()
	defer e.backendMutex.Unlock()

	switch request.Method {
	case "GET":
		responseWriter.WriteHeader(e.backend.membersStatusCode)
		responseWriter.Write([]byte(e.backend.membersJSON))
	case "POST":
		responseWriter.WriteHeader(e.backend.addMembersStatusCode)
		responseWriter.Write([]byte(e.backend.addMembersJSON))
	}
}

func (e *EtcdServer) URL() string {
	return e.server.Addr
}

func (e *EtcdServer) SetMembersReturn(membersJSON string, statusCode int) {
	e.backendMutex.Lock()
	defer e.backendMutex.Unlock()

	e.backend.membersJSON = membersJSON
	e.backend.membersStatusCode = statusCode
}

func (e *EtcdServer) SetAddMemberReturn(memberJSON string, statusCode int) {
	e.backendMutex.Lock()
	defer e.backendMutex.Unlock()

	e.backend.addMembersJSON = memberJSON
	e.backend.addMembersStatusCode = statusCode
}
