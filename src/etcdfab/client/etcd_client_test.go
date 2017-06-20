package client_test

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes/etcdserver"
	"github.com/coreos/etcd/pkg/transport"

	coreosetcdclient "github.com/coreos/etcd/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EtcdClient", func() {
	var (
		etcdServer *etcdserver.EtcdServer

		etcdClient *client.EtcdClient

		logger *fakes.Logger
		cfg    *fakes.Config
	)

	BeforeEach(func() {
		logger = &fakes.Logger{}
		cfg = &fakes.Config{}

		etcdServer = etcdserver.NewEtcdServer(false, "")
		cfg.EtcdClientEndpointsCall.Returns.Endpoints = []string{fmt.Sprintf("%s", etcdServer.URL())}
		cfg.EtcdClientSelfEndpointCall.Returns.Endpoint = fmt.Sprintf("%s", etcdServer.URL())

		// wait for server to start
		time.Sleep(10 * time.Millisecond)

		etcdClient = client.NewEtcdClient(logger)
	})

	AfterEach(func() {
		etcdServer.Exit()
	})

	Describe("Configure", func() {
		Context("when etcdfabConfig.RequireSSL() is false", func() {
			It("configures the etcd client with etcdfab config", func() {
				err := etcdClient.Configure(cfg)
				Expect(err).NotTo(HaveOccurred())

				Expect(cfg.EtcdClientEndpointsCall.CallCount).To(Equal(1))
				Expect(cfg.EtcdClientSelfEndpointCall.CallCount).To(Equal(1))
				Expect(logger.Messages()).To(Equal([]fakes.LoggerMessage{
					{
						Action: "etcd-client.configure.config",
						Data: []lager.Data{
							{
								"endpoints":     []string{fmt.Sprintf("%s", etcdServer.URL())},
								"self-endpoint": fmt.Sprintf("%s", etcdServer.URL()),
							},
						},
					},
				}))
			})
		})

		Context("when etcdfabConfig.RequireSSL() is true", func() {
			var (
				actualTLSInfo transport.TLSInfo
			)

			BeforeEach(func() {
				cfg.RequireSSLCall.Returns.RequireSSL = true
				cfg.CertDirCall.Returns.CertDir = "some/cert/dir"

				client.SetNewTransport(func(tlsInfo transport.TLSInfo) (*http.Transport, error) {
					actualTLSInfo = tlsInfo
					return nil, nil
				})
			})

			AfterEach(func() {
				client.ResetNewTransport()
			})

			It("configures the etcd client with etcdfab config", func() {
				err := etcdClient.Configure(cfg)
				Expect(err).NotTo(HaveOccurred())

				Expect(logger.Messages()).To(Equal([]fakes.LoggerMessage{
					{
						Action: "etcd-client.configure.config",
						Data: []lager.Data{
							{
								"endpoints":     []string{fmt.Sprintf("%s", etcdServer.URL())},
								"self-endpoint": fmt.Sprintf("%s", etcdServer.URL()),
							},
						},
					},
				}))

				Expect(actualTLSInfo).To(Equal(transport.TLSInfo{
					CAFile:         "some/cert/dir/server-ca.crt",
					CertFile:       "some/cert/dir/client.crt",
					KeyFile:        "some/cert/dir/client.key",
					ClientCertAuth: true,
				}))
			})
		})

		Context("failure cases", func() {
			Context("when no endpoints exist", func() {
				BeforeEach(func() {
					cfg.EtcdClientEndpointsCall.Returns.Endpoints = []string{}
				})

				It("returns an error when config does not contain valid information", func() {
					err := etcdClient.Configure(cfg)
					Expect(err).To(MatchError("client: no endpoints available"))
				})
			})

			Context("when newTransport fails", func() {
				BeforeEach(func() {
					cfg.RequireSSLCall.Returns.RequireSSL = true

					client.SetNewTransport(func(tlsInfo transport.TLSInfo) (*http.Transport, error) {
						return nil, errors.New("failed to create new transport")
					})
				})

				AfterEach(func() {
					client.ResetNewTransport()
				})

				It("returns an error when config does not contain valid information", func() {
					err := etcdClient.Configure(cfg)
					Expect(err).To(MatchError("failed to create new transport"))
				})
			})
		})
	})

	Describe("Self", func() {
		BeforeEach(func() {
			cfg.EtcdClientEndpointsCall.Returns.Endpoints = []string{"https://non-existant-server"}
			cfg.EtcdClientSelfEndpointCall.Returns.Endpoint = fmt.Sprintf("%s", etcdServer.URL())

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

			err := etcdClient.Configure(cfg)
			Expect(err).NotTo(HaveOccurred())

		})

		It("returns an etcd client that uses the self endpoint", func() {
			selfEtcdClient, err := etcdClient.Self()
			Expect(err).NotTo(HaveOccurred())

			Expect(selfEtcdClient).ToNot(BeNil())

			_, err = selfEtcdClient.MemberList()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("failure cases", func() {
			BeforeEach(func() {
				client.SetCoreOSEtcdClientNew(func(cfg coreosetcdclient.Config) (coreosetcdclient.Client, error) {
					return nil, errors.New("failed to create etcd client")
				})
			})

			AfterEach(func() {
				client.ResetCoreOSEtcdClientNew()
			})

			It("returns an error when it fails to create the etcd client", func() {
				_, err := etcdClient.Self()
				Expect(err).To(MatchError("failed to create etcd client"))
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

			err := etcdClient.Configure(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns a list of members in the cluster", func() {
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

		Context("when members api list fails", func() {
			BeforeEach(func() {
				etcdServer.SetMembersReturn("", http.StatusInternalServerError)
			})

			It("returns an error", func() {
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
				"id": "some-id-1",
				"peerURLs": [
					"http://some-node-url-1:7001"
				]
			}`, http.StatusCreated)

			err := etcdClient.Configure(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns a list of members in the cluster", func() {
			member, err := etcdClient.MemberAdd("http://some-node-url-1:7001")
			Expect(err).NotTo(HaveOccurred())

			Expect(member).To(Equal(client.Member{
				ID: "some-id-1",
				PeerURLs: []string{
					"http://some-node-url-1:7001",
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

	Describe("MemberRemove", func() {
		BeforeEach(func() {
			etcdServer.SetMembersReturn(`{
				"members": [
					{
						"id": "member-id",
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
			etcdServer.SetRemoveMemberReturn(http.StatusNoContent)

			err := etcdClient.Configure(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes the member from the cluster", func() {
			err := etcdClient.MemberRemove("member-id")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when members api remove fails", func() {
			BeforeEach(func() {
				etcdServer.SetRemoveMemberReturn(http.StatusInternalServerError)
			})

			It("returns an error", func() {
				err := etcdClient.MemberRemove("member-id")
				Expect(err).To(MatchError("client: etcd cluster is unavailable or misconfigured"))
			})
		})
	})

	Describe("Keys", func() {
		BeforeEach(func() {
			err := etcdClient.Configure(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when keys api returns 200", func() {
			BeforeEach(func() {
				etcdServer.SetKeysReturn(http.StatusOK)
			})

			It("does not return an error", func() {
				err := etcdClient.Keys()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when keys api fails", func() {
			BeforeEach(func() {
				etcdServer.SetKeysReturn(http.StatusInternalServerError)
			})

			It("returns an error", func() {
				err := etcdClient.Keys()
				Expect(err).To(MatchError("client: etcd cluster is unavailable or misconfigured"))
			})
		})
	})
})
