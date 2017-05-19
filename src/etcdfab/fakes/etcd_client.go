package fakes

import (
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"
)

type EtcdClient struct {
	ConfigureCall struct {
		CallCount int
		Receives  struct {
			Config config.Config
		}
		Returns struct {
			Error error
		}
	}
	MemberListCall struct {
		CallCount int
		Returns   struct {
			MemberList []client.Member
			Error      error
		}
	}
	MemberAddCall struct {
		CallCount int
		Receives  struct {
			PeerURL string
		}
		Returns struct {
			Member client.Member
			Error  error
		}
	}
}

func (e *EtcdClient) Configure(config config.Config) error {
	e.ConfigureCall.CallCount++
	e.ConfigureCall.Receives.Config = config

	return e.ConfigureCall.Returns.Error
}

func (e *EtcdClient) MemberList() ([]client.Member, error) {
	e.MemberListCall.CallCount++

	return e.MemberListCall.Returns.MemberList, e.MemberListCall.Returns.Error
}

func (e *EtcdClient) MemberAdd(peerURL string) (client.Member, error) {
	e.MemberAddCall.CallCount++
	e.MemberAddCall.Receives.PeerURL = peerURL

	return e.MemberAddCall.Returns.Member, e.MemberAddCall.Returns.Error
}
