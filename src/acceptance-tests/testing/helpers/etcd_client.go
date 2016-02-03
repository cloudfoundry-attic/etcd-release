package helpers

import (
	"acceptance-tests/testing/etcd"

	goetcd "github.com/coreos/go-etcd/etcd"
)

func NewEtcdClient(machines []string) etcd.Client {
	return etcd.NewClient(goetcd.NewClient(machines))
}
