package sync

import (
	"time"
)

const maxSyncCalls = 20

type etcdClient interface {
	Keys() error
}

type Controller struct {
	etcdClient etcdClient
	sleep      func(time.Duration)
}

func NewController(etcdClient etcdClient, sleep func(time.Duration)) Controller {
	return Controller{
		etcdClient: etcdClient,
		sleep:      sleep}
}

func (c Controller) VerifySynced() error {
	var err error
	for i := 0; i < maxSyncCalls; i++ {
		err = c.etcdClient.Keys()
		if err == nil {
			return nil
		}
		c.sleep(1 * time.Second)
	}

	return err
}
