package sync

import (
	"time"
)

const maxSyncCalls = 20

type etcdClient interface {
	Keys() error
}

type Controller struct {
	client etcdClient
	sleep  func(time.Duration)
}

func NewController(client etcdClient, sleep func(time.Duration)) Controller {
	return Controller{client: client, sleep: sleep}
}

func (c Controller) VerifySynced() error {
	var err error
	for i := 0; i < maxSyncCalls; i++ {
		err = c.client.Keys()
		if err == nil {
			return nil
		}
		c.sleep(1 * time.Second)
	}
	return err
}
