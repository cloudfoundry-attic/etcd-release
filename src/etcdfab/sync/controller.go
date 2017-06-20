package sync

import (
	"time"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"

	"code.cloudfoundry.org/lager"
)

const maxSyncCalls = 20

type etcdClient interface {
	Self() (client.EtcdClientInterface, error)
	Keys() error
}

type logger interface {
	Info(string, ...lager.Data)
	Error(string, error, ...lager.Data)
}

type Controller struct {
	etcdClient etcdClient
	logger     logger
	sleep      func(time.Duration)
}

func NewController(etcdClient etcdClient, logger logger, sleep func(time.Duration)) Controller {
	return Controller{
		etcdClient: etcdClient,
		logger:     logger,
		sleep:      sleep,
	}
}

func (c Controller) VerifySynced() error {
	c.logger.Info("sync.verify-synced", lager.Data{
		"max-sync-calls": maxSyncCalls,
	})

	selfEtcdClient, err := c.etcdClient.Self()
	if err != nil {
		return err
	}

	for i := 0; i < maxSyncCalls; i++ {
		c.logger.Info("sync.verify-synced.check-keys", lager.Data{
			"index": i,
		})
		err = selfEtcdClient.Keys()
		if err == nil {
			return nil
		} else {
			c.logger.Error("sync.verify-synced.check-keys.failed", err)
		}
		c.sleep(1 * time.Second)
	}

	return err
}
