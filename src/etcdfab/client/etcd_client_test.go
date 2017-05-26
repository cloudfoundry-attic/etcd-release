package client_test

import (
	"fmt"
	"net/http"
	"os"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes/etcdserver"

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

		etcdServer = etcdserver.NewEtcdServer()
		cfg.EtcdClientEndpointsCall.Returns.Endpoints = []string{fmt.Sprintf("%s", etcdServer.URL())}
		cfg.RequireSSLCall.Returns.RequireSSL = false
	})

	AfterEach(func() {
		etcdServer.Exit()
	})

	Describe("Configure", func() {
		Context("when etcdfabConfig.RequireSSL() is false", func() {
			BeforeEach(func() {
				cfg.RequireSSLCall.Returns.RequireSSL = false
			})

			It("configures the etcd client with etcdfab config", func() {
				etcdClient = client.NewEtcdClient(logger)

				err := etcdClient.Configure(cfg, "")
				Expect(err).NotTo(HaveOccurred())

				Expect(logger.Messages()).To(Equal([]fakes.LoggerMessage{
					{
						Action: "etcd-client.configure.config",
						Data: []lager.Data{
							{
								"endpoints": []string{fmt.Sprintf("%s", etcdServer.URL())},
							},
						},
					},
				}))
			})
		})

		Context("when etcdfabConfig.RequireSSL() is true", func() {
			var (
				certDir string
				//caCert     string
				//clientCert string
				//clientKey  string
			)

			BeforeEach(func() {
				cfg.RequireSSLCall.Returns.RequireSSL = true

				//caCert = "some-ca-cert"
				//clientCert = "some-client-cert"
				//clientKey = "some-client-key"

				//var err error
				//certDir, err = ioutil.TempDir("", "")
				//Expect(err).NotTo(HaveOccurred())
				certDir = "../fixtures"
				if _, err := os.Stat(certDir); os.IsNotExist(err) {
					panic("certDir does not exist")
				}

				//caCertFile := filepath.Join(certDir, "server-ca.crt")
				//clientCertFile := filepath.Join(certDir, "client.crt")
				//clientKeyFile := filepath.Join(certDir, "client.key")

				//err = ioutil.WriteFile(caCertFile, []byte(caCert), os.ModePerm)
				//Expect(err).NotTo(HaveOccurred())
				//err = ioutil.WriteFile(clientCertFile, []byte(clientCert), os.ModePerm)
				//Expect(err).NotTo(HaveOccurred())
				//err = ioutil.WriteFile(clientKeyFile, []byte(clientKey), os.ModePerm)
				//Expect(err).NotTo(HaveOccurred())
			})

			It("configures the etcd client with etcdfab config", func() {
				etcdClient = client.NewEtcdClient(logger)

				err := etcdClient.Configure(cfg, certDir)
				Expect(err).NotTo(HaveOccurred())

				Expect(logger.Messages()).To(Equal([]fakes.LoggerMessage{
					{
						Action: "etcd-client.configure.config",
						Data: []lager.Data{
							{
								"endpoints": []string{fmt.Sprintf("%s", etcdServer.URL())},
							},
						},
					},
				}))
			})
		})

		Context("failure cases", func() {
			BeforeEach(func() {
				cfg.EtcdClientEndpointsCall.Returns.Endpoints = []string{}
				etcdClient = client.NewEtcdClient(logger)
			})

			It("returns an error when config does not contain valid information", func() {
				err := etcdClient.Configure(cfg, "")
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

			err := etcdClient.Configure(cfg, "")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when require ssl is enabled", func() {
			BeforeEach(func() {
				cfg.RequireSSLCall.Returns.RequireSSL = true
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

			etcdClient = client.NewEtcdClient(logger)

			err := etcdClient.Configure(cfg, "")
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
})
