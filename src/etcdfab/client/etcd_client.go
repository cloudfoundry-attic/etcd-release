package client

import (
	"context"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/coreos/etcd/pkg/transport"

	coreosetcdclient "github.com/coreos/etcd/client"
)

type EtcdClientInterface interface {
	MemberList() ([]Member, error)
	MemberAdd(string) (Member, error)
	Keys() error
}

type EtcdClient struct {
	coreosEtcdClient coreosetcdclient.Client
	clientConfig     coreosetcdclient.Config
	selfEndpoint     string

	logger logger
}

type Member struct {
	ID         string
	Name       string
	PeerURLs   []string
	ClientURLs []string
}

type Config interface {
	EtcdClientEndpoints() []string
	EtcdClientSelfEndpoint() string
	RequireSSL() bool
	CertDir() string
}

type logger interface {
	Info(string, ...lager.Data)
	Error(string, error, ...lager.Data)
}

var newTransport = transport.NewTransport
var coreOSEtcdClientNew = coreosetcdclient.New

func NewEtcdClient(logger logger) *EtcdClient {
	return &EtcdClient{
		logger: logger,
	}
}

func (e *EtcdClient) Configure(etcdfabConfig Config) error {
	endpoints := etcdfabConfig.EtcdClientEndpoints()
	e.selfEndpoint = etcdfabConfig.EtcdClientSelfEndpoint()
	e.logger.Info("etcd-client.configure.config", lager.Data{
		"endpoints":     endpoints,
		"self-endpoint": e.selfEndpoint,
	})

	tns := coreosetcdclient.DefaultTransport

	var err error
	if etcdfabConfig.RequireSSL() {
		caCertFile := filepath.Join(etcdfabConfig.CertDir(), "server-ca.crt")
		clientCertFile := filepath.Join(etcdfabConfig.CertDir(), "client.crt")
		clientKeyFile := filepath.Join(etcdfabConfig.CertDir(), "client.key")

		tlsInfo := transport.TLSInfo{
			CAFile:         caCertFile,
			CertFile:       clientCertFile,
			KeyFile:        clientKeyFile,
			ClientCertAuth: etcdfabConfig.RequireSSL(),
		}

		tns, err = newTransport(tlsInfo)
		if err != nil {
			return err
		}
	}

	e.clientConfig = coreosetcdclient.Config{
		Endpoints:               endpoints,
		Transport:               tns,
		HeaderTimeoutPerRequest: time.Second,
	}
	e.coreosEtcdClient, err = coreosetcdclient.New(e.clientConfig)
	if err != nil {
		return err
	}

	return nil
}

func (e *EtcdClient) Self() (EtcdClientInterface, error) {
	var selfEtcdClient = &EtcdClient{}
	*selfEtcdClient = *e

	selfEtcdClient.clientConfig.Endpoints = []string{e.selfEndpoint}

	var err error
	selfEtcdClient.coreosEtcdClient, err = coreOSEtcdClientNew(selfEtcdClient.clientConfig)
	if err != nil {
		return nil, err
	}

	return selfEtcdClient, nil
}

func (e *EtcdClient) MemberList() ([]Member, error) {
	membersAPI := coreosetcdclient.NewMembersAPI(e.coreosEtcdClient)
	memberList, err := membersAPI.List(context.Background())
	if err != nil {
		return []Member{}, err
	}

	var members []Member
	for _, m := range memberList {
		members = append(members, Member{
			ID:         m.ID,
			Name:       m.Name,
			PeerURLs:   m.PeerURLs,
			ClientURLs: m.ClientURLs,
		})
	}

	return members, nil
}

func (e *EtcdClient) MemberAdd(peerURL string) (Member, error) {
	membersAPI := coreosetcdclient.NewMembersAPI(e.coreosEtcdClient)
	m, err := membersAPI.Add(context.Background(), peerURL)
	if err != nil {
		return Member{}, err
	}
	return Member{
		ID:         m.ID,
		Name:       m.Name,
		PeerURLs:   m.PeerURLs,
		ClientURLs: m.ClientURLs,
	}, nil
}

func (e *EtcdClient) MemberRemove(memberID string) error {
	membersAPI := coreosetcdclient.NewMembersAPI(e.coreosEtcdClient)
	err := membersAPI.Remove(context.Background(), memberID)
	if err != nil {
		return err
	}
	return nil
}

func (e *EtcdClient) Keys() error {
	keysAPI := coreosetcdclient.NewKeysAPI(e.coreosEtcdClient)
	_, err := keysAPI.Get(context.Background(), "", &coreosetcdclient.GetOptions{})
	return err
}
