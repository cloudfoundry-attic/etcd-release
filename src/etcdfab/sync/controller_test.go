package sync_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/sync"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller", func() {
	var (
		etcdClient *fakes.EtcdClient

		syncController sync.Controller

		sleepFunc      func(time.Duration)
		sleepDuration  time.Duration
		sleepCallCount int
	)

	BeforeEach(func() {
		etcdClient = &fakes.EtcdClient{}
		sleepFunc = func(duration time.Duration) {
			sleepCallCount++
			sleepDuration = duration
		}

		syncController = sync.NewController(etcdClient, sleepFunc)
	})

	AfterEach(func() {
		sleepCallCount = 0
		sleepDuration = 0
	})

	Describe("VerifySynced", func() {
		Context("when etcdClient.Keys eventually returns no error", func() {
			BeforeEach(func() {
				etcdClient.KeysCall.Stub = func() error {
					if etcdClient.KeysCall.CallCount >= 5 {
						return nil
					}

					return errors.New("not synced")
				}
			})

			It("returns no error", func() {
				err := syncController.VerifySynced()
				Expect(err).NotTo(HaveOccurred())

				Expect(etcdClient.KeysCall.CallCount).To(Equal(5))
				Expect(sleepDuration).To(Equal(1 * time.Second))
				Expect(sleepCallCount).To(Equal(4))
			})
		})

		Context("when etcdClient.Keys never syncs", func() {
			BeforeEach(func() {
				etcdClient.KeysCall.Returns.Error = errors.New("never syncs")
			})

			It("returns the error", func() {
				err := syncController.VerifySynced()
				Expect(err).To(MatchError("never syncs"))

				Expect(etcdClient.KeysCall.CallCount).To(Equal(20))
				Expect(sleepCallCount).To(Equal(20))
			})
		})
	})
})
