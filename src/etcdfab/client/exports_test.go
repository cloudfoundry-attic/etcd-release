package client

import (
	"net/http"

	"github.com/coreos/etcd/pkg/transport"

	coreosetcdclient "github.com/coreos/etcd/client"
)

func SetCoreOSEtcdClientNew(f func(cfg coreosetcdclient.Config) (coreosetcdclient.Client, error)) {
	coreOSEtcdClientNew = f
}

func ResetCoreOSEtcdClientNew() {
	coreOSEtcdClientNew = coreosetcdclient.New
}

func SetNewTransport(f func(transport.TLSInfo) (*http.Transport, error)) {
	newTransport = f
}

func ResetNewTransport() {
	newTransport = transport.NewTransport
}
