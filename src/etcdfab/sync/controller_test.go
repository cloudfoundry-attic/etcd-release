package sync_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/sync"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller", func() {
	var (
		etcdClient *fakes.EtcdClient
		logger     *fakes.Logger

		syncController sync.Controller

		sleepFunc      func(time.Duration)
		sleepDuration  time.Duration
		sleepCallCount int
	)

	BeforeEach(func() {
		etcdClient = &fakes.EtcdClient{}
		logger = &fakes.Logger{}
		sleepFunc = func(duration time.Duration) {
			sleepCallCount++
			sleepDuration = duration
		}

		syncController = sync.NewController(etcdClient, logger, sleepFunc)
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
				Expect(logger.Messages()).To(ConsistOf([]fakes.LoggerMessage{
					{
						Action: "sync.verify-synced",
						Data: []lager.Data{{
							"max-sync-calls": 20,
						}},
					},
					{
						Action: "sync.verify-synced.check-keys",
						Data: []lager.Data{{
							"index": 0,
						}},
					},
					{
						Action: "sync.verify-synced.check-keys.failed",
						Error:  errors.New("not synced"),
					},
					{
						Action: "sync.verify-synced.check-keys",
						Data: []lager.Data{{
							"index": 1,
						}},
					},
					{
						Action: "sync.verify-synced.check-keys.failed",
						Error:  errors.New("not synced"),
					},
					{
						Action: "sync.verify-synced.check-keys",
						Data: []lager.Data{{
							"index": 2,
						}},
					},
					{
						Action: "sync.verify-synced.check-keys.failed",
						Error:  errors.New("not synced"),
					},
					{
						Action: "sync.verify-synced.check-keys",
						Data: []lager.Data{{
							"index": 3,
						}},
					},
					{
						Action: "sync.verify-synced.check-keys.failed",
						Error:  errors.New("not synced"),
					},
					{
						Action: "sync.verify-synced.check-keys",
						Data: []lager.Data{{
							"index": 4,
						}},
					},
				}))
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
