package sync

import (
	"time"

	"code.cloudfoundry.org/lager"
)

const maxSyncCalls = 20

type etcdClient interface {
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
	var err error
	c.logger.Info("sync.verify-synced", lager.Data{
		"max-sync-calls": maxSyncCalls,
	})
	for i := 0; i < maxSyncCalls; i++ {
		c.logger.Info("sync.verify-synced.check-keys", lager.Data{
			"index": i,
		})
		err = c.etcdClient.Keys()
		if err == nil {
			return nil
		} else {
			c.logger.Error("sync.verify-synced.check-keys.failed", err)
		}
		c.sleep(1 * time.Second)
	}

	return err
}
