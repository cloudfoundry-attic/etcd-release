package fakes

import (
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/cluster"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"
)

type ClusterController struct {
	GetInitialClusterStateCall struct {
		CallCount int
		Receives  struct {
			Config config.Config
		}
		Returns struct {
			InitialClusterState cluster.InitialClusterState
			Error               error
		}
	}
}

func (c *ClusterController) GetInitialClusterState(etcdfabConfig config.Config) (cluster.InitialClusterState, error) {
	c.GetInitialClusterStateCall.CallCount++
	c.GetInitialClusterStateCall.Receives.Config = etcdfabConfig

	return c.GetInitialClusterStateCall.Returns.InitialClusterState, c.GetInitialClusterStateCall.Returns.Error
}
