package fakes

import (
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"
)

type EtcdClient struct {
	ConfigureCall struct {
		CallCount int
		Receives  struct {
			Config client.Config
		}
		Returns struct {
			Error error
		}
	}
	SelfCall struct {
		CallCount int
		Returns   struct {
			EtcdClient client.EtcdClientInterface
			Error      error
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
	MemberRemoveCall struct {
		CallCount int
		Receives  struct {
			MemberID string
		}
		Returns struct {
			Error error
		}
	}
	KeysCall struct {
		CallCount int
		Stub      func() error
		Returns   struct {
			Error error
		}
	}
}

func (e *EtcdClient) Configure(config client.Config) error {
	e.ConfigureCall.CallCount++
	e.ConfigureCall.Receives.Config = config

	return e.ConfigureCall.Returns.Error
}

func (e *EtcdClient) Self() (client.EtcdClientInterface, error) {
	e.SelfCall.CallCount++

	return e.SelfCall.Returns.EtcdClient, e.SelfCall.Returns.Error
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

func (e *EtcdClient) MemberRemove(memberID string) error {
	e.MemberRemoveCall.CallCount++
	e.MemberRemoveCall.Receives.MemberID = memberID

	return e.MemberRemoveCall.Returns.Error
}

func (e *EtcdClient) Keys() error {
	e.KeysCall.CallCount++

	if e.KeysCall.Stub != nil {
		return e.KeysCall.Stub()
	}

	return e.KeysCall.Returns.Error
}
