package client

import (
	"context"
	"time"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"
	coreosetcdclient "github.com/coreos/etcd/client"
)

type EtcdClient struct {
	coreosEtcdClient coreosetcdclient.Client
	membersAPI       coreosetcdclient.MembersAPI
}

type Member struct {
	ID         string
	Name       string
	PeerURLs   []string
	ClientURLs []string
}

func NewEtcdClient() *EtcdClient {
	return &EtcdClient{}
}

func (e *EtcdClient) Configure(etcdfabConfig config.Config) error {
	cfg := coreosetcdclient.Config{
		Endpoints:               etcdfabConfig.Etcd.Machines,
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
