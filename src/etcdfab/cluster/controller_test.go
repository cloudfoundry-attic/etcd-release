package cluster_test

import (
	"errors"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/cluster"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller", func() {
	Describe("GetInitialClusterState", func() {
		var (
			etcdClient *fakes.EtcdClient
			logger     *fakes.Logger

			sleep                 func(time.Duration)
			sleepCallCount        int
			sleepReceivedDuration time.Duration

			controller cluster.Controller
		)

		BeforeEach(func() {
			etcdClient = &fakes.EtcdClient{}
			logger = &fakes.Logger{}
			sleep = func(duration time.Duration) {
				sleepCallCount++
				sleepReceivedDuration = duration
			}

			controller = cluster.NewController(etcdClient, logger, sleep)
		})

		AfterEach(func() {
			sleepCallCount = 0
		})

		Context("when no prior cluster members", func() {
			It("returns state new and the itself as the member list", func() {
				initialClusterState, err := controller.GetInitialClusterState(config.Config{
					Node: config.Node{
						Name:       "some_name",
						Index:      0,
						ExternalIP: "some-external-ip",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(initialClusterState.Members).To(Equal("some-name-0=http://some-external-ip:7001"))
				Expect(initialClusterState.State).To(Equal("new"))
				Expect(etcdClient.MemberListCall.CallCount).To(Equal(1))
				Expect(etcdClient.MemberAddCall.CallCount).To(Equal(0))
				Expect(logger.Messages()).To(ConsistOf([]fakes.LoggerMessage{
					{
						Action: "cluster.get-initial-cluster-state.member-list",
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list.no-members-found",
					},
					{
						Action: "cluster.get-initial-cluster-state.return",
						Data: []lager.Data{
							{
								"initial_cluster_state": cluster.InitialClusterState{
									Members: "some-name-0=http://some-external-ip:7001",
									State:   "new",
								},
							},
						},
					},
				}))
			})

			It("retries finding cluster member five times", func() {
				etcdClient.MemberListCall.Returns.Error = errors.New("failed to call member list")
				_, err := controller.GetInitialClusterState(config.Config{
					Node: config.Node{
						Name:       "some_name",
						Index:      0,
						ExternalIP: "some-external-ip",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(etcdClient.MemberListCall.CallCount).To(Equal(5))
				Expect(sleepCallCount).To(Equal(5))
				Expect(sleepReceivedDuration).To(Equal(1 * time.Second))
				Expect(logger.Messages()).To(ConsistOf([]fakes.LoggerMessage{
					{
						Action: "cluster.get-initial-cluster-state.member-list",
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list.failed",
						Error:  errors.New("failed to call member list"),
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list",
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list.failed",
						Error:  errors.New("failed to call member list"),
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list",
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list.failed",
						Error:  errors.New("failed to call member list"),
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list",
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list.failed",
						Error:  errors.New("failed to call member list"),
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list",
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list.failed",
						Error:  errors.New("failed to call member list"),
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list.no-members-found",
					},
					{
						Action: "cluster.get-initial-cluster-state.return",
						Data: []lager.Data{
							{
								"initial_cluster_state": cluster.InitialClusterState{
									Members: "some-name-0=http://some-external-ip:7001",
									State:   "new",
								},
							},
						},
					},
				}))
			})
		})

		Context("when prior cluster members exist", func() {
			BeforeEach(func() {
				etcdClient.MemberListCall.Returns.MemberList = []client.Member{
					{
						Name:     "some-prior-node",
						PeerURLs: []string{"http://some-peer-url:7001"},
					},
				}
			})

			It("returns state existing and all prior members plus itself as the member list", func() {
				initialClusterState, err := controller.GetInitialClusterState(config.Config{
					Node: config.Node{
						Name:       "some_name",
						Index:      0,
						ExternalIP: "some-external-ip",
					},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(etcdClient.MemberListCall.CallCount).To(Equal(1))
				Expect(etcdClient.MemberAddCall.CallCount).To(Equal(1))
				Expect(etcdClient.MemberAddCall.Receives.PeerURL).To(Equal("http://some-external-ip:7001"))
				Expect(sleepCallCount).To(Equal(1))
				Expect(sleepReceivedDuration).To(Equal(2 * time.Second))

				Expect(initialClusterState.Members).To(Equal("some-prior-node=http://some-peer-url:7001,some-name-0=http://some-external-ip:7001"))
				Expect(initialClusterState.State).To(Equal("existing"))
				Expect(logger.Messages()).To(ConsistOf([]fakes.LoggerMessage{
					{
						Action: "cluster.get-initial-cluster-state.member-list",
					},
					{
						Action: "cluster.get-initial-cluster-state.member-list.members",
						Data: []lager.Data{
							{
								"prior_members": []client.Member{
									{
										Name:     "some-prior-node",
										PeerURLs: []string{"http://some-peer-url:7001"},
									},
								},
							},
						},
					},
					{
						Action: "cluster.get-initial-cluster-state.return",
						Data: []lager.Data{
							{
								"initial_cluster_state": cluster.InitialClusterState{
									Members: "some-prior-node=http://some-peer-url:7001,some-name-0=http://some-external-ip:7001",
									State:   "existing",
								},
							},
						},
					},
				}))
			})

			Context("when MemberAdd fails", func() {
				BeforeEach(func() {
					etcdClient.MemberAddCall.Returns.Error = errors.New("failed to call member add")
				})

				It("returns the error and logs a helpful message", func() {
					_, err := controller.GetInitialClusterState(config.Config{
						Node: config.Node{
							Name:       "some_name",
							Index:      0,
							ExternalIP: "some-external-ip",
						},
					})
					Expect(err).To(MatchError("failed to call member add"))
				})
			})

			Context("when this node is part of the prior cluster members list", func() {
				BeforeEach(func() {
					etcdClient.MemberListCall.Returns.MemberList = []client.Member{
						{
							Name:     "some-prior-node",
							PeerURLs: []string{"http://some-peer-url:7001"},
						},
						{
							Name:     "some-name-0",
							PeerURLs: []string{"http://some-external-ip:7001"},
						},
					}
				})

				It("returns state existing and all prior members as the member list", func() {
					initialClusterState, err := controller.GetInitialClusterState(config.Config{
						Node: config.Node{
							Name:       "some_name",
							Index:      0,
							ExternalIP: "some-external-ip",
						},
					})
					Expect(err).NotTo(HaveOccurred())

					Expect(etcdClient.MemberListCall.CallCount).To(Equal(1))
					Expect(etcdClient.MemberAddCall.CallCount).To(Equal(0))

					Expect(initialClusterState.Members).To(Equal("some-prior-node=http://some-peer-url:7001,some-name-0=http://some-external-ip:7001"))
					Expect(initialClusterState.State).To(Equal("existing"))
					Expect(logger.Messages()).To(ConsistOf([]fakes.LoggerMessage{
						{
							Action: "cluster.get-initial-cluster-state.member-list",
						},
						{
							Action: "cluster.get-initial-cluster-state.member-list.members",
							Data: []lager.Data{
								{
									"prior_members": []client.Member{
										{
											Name:     "some-prior-node",
											PeerURLs: []string{"http://some-peer-url:7001"},
										},
										{
											Name:     "some-name-0",
											PeerURLs: []string{"http://some-external-ip:7001"},
										},
									},
								},
							},
						},
						{
							Action: "cluster.get-initial-cluster-state.return",
							Data: []lager.Data{
								{
									"initial_cluster_state": cluster.InitialClusterState{
										Members: "some-prior-node=http://some-peer-url:7001,some-name-0=http://some-external-ip:7001",
										State:   "existing",
									},
								},
							},
						},
					}))
				})

			})
		})

		Context("when the cluster requires TLS", func() {
			It("returns the initial list of members with the correct protcol and dns suffix for this node", func() {
				initialClusterState, err := controller.GetInitialClusterState(config.Config{
					Node: config.Node{
						Name:       "some_name",
						Index:      0,
						ExternalIP: "some-external-ip",
					},
					Etcd: config.Etcd{
						PeerRequireSSL:         true,
						RequireSSL:             true,
						AdvertiseURLsDNSSuffix: "some-dns-suffix",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(initialClusterState.Members).To(Equal("some-name-0=https://some-name-0.some-dns-suffix:7001"))
				Expect(initialClusterState.State).To(Equal("new"))
			})
		})
	})
})
