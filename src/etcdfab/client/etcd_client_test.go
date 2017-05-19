package client_test

import (
	"net/http"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes/etcdserver"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EtcdClient", func() {
	var (
		etcdServer *etcdserver.EtcdServer

		etcdClient *client.EtcdClient

		cfg config.Config
	)

	BeforeEach(func() {
		etcdServer = etcdserver.NewEtcdServer()
		cfg = config.Config{
			Etcd: config.Etcd{
				Machines: []string{
					etcdServer.URL(),
				},
			},
		}

		etcdClient = client.NewEtcdClient()

		err := etcdClient.Configure(cfg)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Configure", func() {
		It("configures the etcd client with etcdfab config", func() {
			etcdClient = client.NewEtcdClient()

			err := etcdClient.Configure(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("failure cases", func() {
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
						"id": "some-id",
						"name": "some-node-1",
						"peerURLs": [
							"http://some-node-url:7001"
						],
						"clientURLs": [
							"http://some-node-url:4001"
						]
					}
				]
			}`, http.StatusOK)
		})

		Context("when require ssl is enabled", func() {
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
				"id": "some-id",
				"peerURLs": [
					"http://some-node-url:7001"
				]
			}`, http.StatusCreated)
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
