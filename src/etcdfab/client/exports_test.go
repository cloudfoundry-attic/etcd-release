package client

import (
	"net/http"

	"github.com/coreos/etcd/pkg/transport"
)

func SetNewTransport(f func(transport.TLSInfo) (*http.Transport, error)) {
	newTransport = f
}

func ResetNewTransport() {
	newTransport = transport.NewTransport
}
