package client

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"
	coreosetcdclient "github.com/coreos/etcd/client"
)

type EtcdClient struct {
	coreosEtcdClient coreosetcdclient.Client
	membersAPI       coreosetcdclient.MembersAPI

	logger logger
}

type Member struct {
	ID         string
	Name       string
	PeerURLs   []string
	ClientURLs []string
}

type logger interface {
	Info(string, ...lager.Data)
	Error(string, error, ...lager.Data)
}

func NewEtcdClient(logger logger) *EtcdClient {
	return &EtcdClient{
		logger: logger,
	}
}

func (e *EtcdClient) Configure(etcdfabConfig config.Config) error {
	endpoints := []string{}
	if etcdfabConfig.Etcd.RequireSSL || etcdfabConfig.Etcd.PeerRequireSSL {
		endpoints = []string{fmt.Sprintf("https://%s:4001", etcdfabConfig.Etcd.AdvertiseURLsDNSSuffix)}
	} else {
		for _, machineIP := range etcdfabConfig.Etcd.Machines {
			endpoints = append(endpoints, fmt.Sprintf("http://%s:4001", machineIP))
		}
	}
	e.logger.Info("etcd-client.configure.config", lager.Data{
		"endpoints": endpoints,
	})

	cfg := coreosetcdclient.Config{
		Endpoints:               endpoints,
		Transport:               coreosetcdclient.DefaultTransport,
		HeaderTimeoutPerRequest: time.Second,
	}
	coreosEtcdClient, err := coreosetcdclient.New(cfg)
	if err != nil {
		return err
	}

	membersAPI := coreosetcdclient.NewMembersAPI(coreosEtcdClient)

	e.coreosEtcdClient = coreosEtcdClient
	e.membersAPI = membersAPI

	return nil
}

func (e *EtcdClient) MemberList() ([]Member, error) {
	memberList, err := e.membersAPI.List(context.Background())
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
	m, err := e.membersAPI.Add(context.Background(), peerURL)
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
