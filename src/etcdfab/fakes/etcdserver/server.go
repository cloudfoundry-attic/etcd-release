package etcdserver

import (
	"net/http"
	"net/http/httptest"
	"sync"
)

type EtcdServer struct {
	server  *httptest.Server
	backend *etcdBackend

	backendMutex sync.Mutex
}

type etcdBackend struct {
	membersJSON          string
	membersStatusCode    int
	addMembersJSON       string
	addMembersStatusCode int
}

func NewEtcdServer() *EtcdServer {
	etcdServer := &EtcdServer{
		backend: &etcdBackend{},
	}
	etcdServer.server = httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		etcdServer.ServeHTTP(responseWriter, request)
	}))
	etcdServer.Reset()

	return etcdServer
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

	if request.Method == "GET" {
		responseWriter.WriteHeader(e.backend.membersStatusCode)
		responseWriter.Write([]byte(e.backend.membersJSON))
	} else if request.Method == "POST" {
		responseWriter.WriteHeader(e.backend.addMembersStatusCode)
		responseWriter.Write([]byte(e.backend.addMembersJSON))
	}
}

func (e *EtcdServer) URL() string {
	return e.server.URL
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
