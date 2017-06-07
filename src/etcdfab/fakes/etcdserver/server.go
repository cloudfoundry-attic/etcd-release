package etcdserver

import (
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
)

type EtcdServer struct {
	server *httptest.Server

	backend *etcdBackend

	backendMutex sync.Mutex
}

type etcdBackend struct {
	membersJSON         string
	membersStatusCode   int
	addMemberJSON       string
	addMemberStatusCode int
	keysStatusCode      int
}

func NewEtcdServer() *EtcdServer {
	etcdServer := &EtcdServer{
		backend: &etcdBackend{},
	}
	etcdServer.Reset()

	listener, err := net.Listen("tcp", "127.0.0.1:4001")
	if err != nil {
		panic(err)
	}

	etcdServer.server = &httptest.Server{
		Listener: listener,
		Config: &http.Server{
			Handler: http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
				etcdServer.ServeHTTP(responseWriter, request)
			}),
		},
	}
	etcdServer.server.Start()

	return etcdServer
}

func (e *EtcdServer) Exit() {
	e.server.Close()
}

func (e *EtcdServer) Reset() {
	e.backend = &etcdBackend{
		membersStatusCode:   http.StatusOK,
		addMemberStatusCode: http.StatusCreated,
	}
}

func (e *EtcdServer) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	switch request.URL.Path {
	case "/v2/members":
		e.handleMembers(responseWriter, request)
	case "/v2/keys":
		e.handleKeys(responseWriter, request)
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
		responseWriter.WriteHeader(e.backend.addMemberStatusCode)
		responseWriter.Write([]byte(e.backend.addMemberJSON))
	}
}

func (e *EtcdServer) handleKeys(responseWriter http.ResponseWriter, request *http.Request) {
	e.backendMutex.Lock()
	defer e.backendMutex.Unlock()

	switch request.Method {
	case "GET":
		responseWriter.WriteHeader(e.backend.keysStatusCode)
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

	e.backend.addMemberJSON = memberJSON
	e.backend.addMemberStatusCode = statusCode
}

func (e *EtcdServer) SetKeysReturn(status int) {
	e.backendMutex.Lock()
	defer e.backendMutex.Unlock()

	e.backend.keysStatusCode = status
}
