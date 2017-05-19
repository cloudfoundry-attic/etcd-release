package cluster

import (
	"fmt"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"
)

type InitialClusterState struct {
	Members string
	State   string
}

type Controller struct {
	etcdClient etcdClient
	logger     logger
	sleep      func(time.Duration)
}

type etcdClient interface {
	MemberList() ([]client.Member, error)
	MemberAdd(string) (client.Member, error)
}

type logger interface {
	Info(string, ...lager.Data)
	Error(string, error, ...lager.Data)
}

func NewController(etcdClient etcdClient, logger logger, sleep func(time.Duration)) Controller {
	return Controller{
		etcdClient: etcdClient,
		logger:     logger,
		sleep:      sleep,
	}
}

func (c Controller) GetInitialClusterState(etcdfabConfig config.Config) (InitialClusterState, error) {
	var priorMemberList []client.Member
	for i := 0; i < 5; i++ {
		c.logger.Info("cluster.get-initial-cluster-state.member-list")
		var err error
		priorMemberList, err = c.etcdClient.MemberList()
		if err != nil {
			c.logger.Error("cluster.get-initial-cluster-state.member-list.failed", err)
			c.sleep(1 * time.Second)
			continue
		}

		break
	}

	if len(priorMemberList) == 0 {
		c.logger.Info("cluster.get-initial-cluster-state.member-list.no-members-found")
	}

	initialCluster := InitialClusterState{
		State: "new",
	}
	var members []string
	selfIsPartOfPriorMembers := false
	if len(priorMemberList) > 0 {
		c.logger.Info("cluster.get-initial-cluster-state.member-list.members", lager.Data{
			"prior_members": priorMemberList,
		})
		initialCluster.State = "existing"
		for _, member := range priorMemberList {
			members = append(members, fmt.Sprintf("%s=%s", member.Name, member.PeerURLs[0]))
			if member.PeerURLs[0] == etcdfabConfig.AdvertisePeerURL() {
				selfIsPartOfPriorMembers = true
			}
		}
	}

	if !selfIsPartOfPriorMembers {
		if len(priorMemberList) > 0 {
			_, err := c.etcdClient.MemberAdd(etcdfabConfig.AdvertisePeerURL())
			if err != nil {
				return InitialClusterState{}, err
			}
			c.sleep(2 * time.Second)
		}
		members = append(members, fmt.Sprintf("%s=%s", etcdfabConfig.NodeName(), etcdfabConfig.AdvertisePeerURL()))
	}

	initialCluster.Members = strings.Join(members, ",")

	c.logger.Info("cluster.get-initial-cluster-state.return", lager.Data{
		"initial_cluster_state": initialCluster,
	})
	return initialCluster, nil
}
