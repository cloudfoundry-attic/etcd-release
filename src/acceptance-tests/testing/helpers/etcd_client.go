package helpers

import (
	"acceptance-tests/testing/testconsumer/etcd"

	goetcd "github.com/coreos/go-etcd/etcd"
)

func NewEtcdClient(machines []string) etcd.Client {
	client := goetcd.NewClient(machines)
	client.SetConsistency(goetcd.STRONG_CONSISTENCY)

	return etcd.NewClient(client)
}

func NewEtcdTLSClient(machines []string, certFile, keyFile, caCertFile string) (etcd.Client, error) {
	client, err := goetcd.NewTLSClient(machines, certFile, keyFile, caCertFile)
	if err != nil {
		return etcd.Client{}, err
	}
	client.SetConsistency(goetcd.STRONG_CONSISTENCY)

	return etcd.NewClient(client), nil
}
