package client_test

import (
	"net/http"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes/etcdserver"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EtcdClient", func() {
	var (
		etcdServer *etcdserver.EtcdServer

		etcdClient *client.EtcdClient
		logger     *fakes.Logger

		cfg config.Config
	)

	BeforeEach(func() {
		etcdServer = etcdserver.NewEtcdServer("127.0.0.1:4001")
		logger = &fakes.Logger{}
		cfg = config.Config{
			Etcd: config.Etcd{
				Machines: []string{
					"127.0.0.1",
				},
			},
		}
	})

	AfterEach(func() {
		err := etcdServer.Exit()
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Configure", func() {
		Context("when etcd is non tls", func() {
			It("configures the etcd client with etcdfab config", func() {
				etcdClient = client.NewEtcdClient(logger)

				err := etcdClient.Configure(cfg)
				Expect(err).NotTo(HaveOccurred())

				Expect(logger.Messages()).To(Equal([]fakes.LoggerMessage{
					{
						Action: "etcd-client.configure.config",
						Data: []lager.Data{
							{
								"endpoints": []string{"http://127.0.0.1:4001"},
							},
						},
					},
				}))
			})
		})

		Context("when etcd is tls", func() {
			BeforeEach(func() {
				cfg = config.Config{
					Etcd: config.Etcd{
						RequireSSL:             true,
						PeerRequireSSL:         true,
						AdvertiseURLsDNSSuffix: "etcd.service.cf.internal",
					},
				}
			})

			It("configures the etcd client with etcdfab config", func() {
				etcdClient = client.NewEtcdClient(logger)

				err := etcdClient.Configure(cfg)
				Expect(err).NotTo(HaveOccurred())

				Expect(logger.Messages()).To(Equal([]fakes.LoggerMessage{
					{
						Action: "etcd-client.configure.config",
						Data: []lager.Data{
							{
								"endpoints": []string{"https://etcd.service.cf.internal:4001"},
							},
						},
					},
				}))
			})
		})

		Context("failure cases", func() {
			BeforeEach(func() {
				etcdClient = client.NewEtcdClient(logger)

				err := etcdClient.Configure(cfg)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns an error when config does not contain valid information", func() {
				err := etcdClient.Configure(config.Config{})
				Expect(err).To(MatchError("client: no endpoints available"))
			})
		})
	})

	Describe("MemberList", func() {
		BeforeEach(func() {
			etcdServer.SetMembersReturn(`{
				"members": [
					{
						"id": "some-id", "name": "some-node-1", "peerURLs": [
							"http://some-node-url:7001"
						],
						"clientURLs": [
							"http://some-node-url:4001"
						]
					}
				]
			}`, http.StatusOK)

			etcdClient = client.NewEtcdClient(logger)

			err := etcdClient.Configure(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when require ssl is enabled", func() {
			FIt("returns a list of members in the cluster", func() {
				members, err := etcdClient.MemberList()
				Expect(err).NotTo(HaveOccurred())
				Expect(members).To(Equal([]client.Member{
					{
						ID:   "some-id",
						Name: "some-node-1",
						PeerURLs: []string{
							"http://some-node-url:7001",
						},
						ClientURLs: []string{
							"http://some-node-url:4001",
						},
					},
				}))
			})
		})

		Context("when members api list fails", func() {
			BeforeEach(func() {
				etcdServer.SetMembersReturn("", http.StatusInternalServerError)
			})

			FIt("returns an error", func() {
				_, err := etcdClient.MemberList()
				Expect(err).To(MatchError("client: etcd cluster is unavailable or misconfigured"))
			})
		})
	})

	Describe("MemberAdd", func() {
		BeforeEach(func() {
			etcdServer.SetMembersReturn(`{
				"members": [
					{
						"id": "some-id-2",
						"name": "some-node-2",
						"peerURLs": [
							"http://some-node-url-2:7001"
						],
						"clientURLs": [
							"http://some-node-url-2:4001"
						]
					}
				]
			}`, http.StatusOK)
			etcdServer.SetAddMemberReturn(`{
				"id": "some-id",
				"peerURLs": [
					"http://some-node-url:7001"
				]
			}`, http.StatusCreated)

			etcdClient = client.NewEtcdClient(logger)

			err := etcdClient.Configure(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns a list of members in the cluster", func() {
			members, err := etcdClient.MemberAdd("http://some-other-node-url:7001")
			Expect(err).NotTo(HaveOccurred())

			Expect(members).To(Equal(client.Member{
				ID: "some-id",
				PeerURLs: []string{
					"http://some-node-url:7001",
				},
			}))
		})

		Context("when members api add fails", func() {
			BeforeEach(func() {
				etcdServer.SetAddMemberReturn("", http.StatusInternalServerError)
			})

			It("returns an error", func() {
				_, err := etcdClient.MemberAdd("http://fake-peer-url:111")
				Expect(err).To(MatchError("client: etcd cluster is unavailable or misconfigured"))
			})
		})
	})
})
