package deploy_test

import (
	"acceptance-tests/testing/bosh"
	"acceptance-tests/testing/destiny"
	"acceptance-tests/testing/helpers"

	"github.com/coreos/go-etcd/etcd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scaling down instances", func() {
	var (
		manifest       destiny.Manifest
		testKey        string
		testValue      string
		etcdClient     *etcd.Client
		etcdClientURLs []string
	)

	BeforeEach(func() {
		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		testKey = "etcd-key-" + guid
		testValue = "etcd-value-" + guid

		manifest, err = helpers.DeployEtcdWithInstanceCount(3, client, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
			{"running"},
			{"running"},
		}))
	})

	AfterEach(func() {
		err := client.DeleteDeployment(manifest.Name)
		Expect(err).NotTo(HaveOccurred())
	})

	It("scales from 3 to 1 nodes", func() {
		By("scaling down to 1 node", func() {
			manifest.Jobs[0], manifest.Properties = destiny.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 1)

			members := manifest.EtcdMembers()
			Expect(members).To(HaveLen(1))

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			err = client.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
			}))
		})

		By("instantiating a etcd client connection", func() {
			for _, elem := range manifest.Properties.Etcd.Machines {
				etcdClientURLs = append(etcdClientURLs, "http://"+elem+":4001")
			}

			etcdClient = etcd.NewClient(etcdClientURLs)
		})

		By("setting a persistent value", func() {
			response, err := etcdClient.Create(testKey, testValue, 6000)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
		})

		By("reading the value from etcd", func() {
			response, err := etcdClient.Get(testKey, false, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Node.Value).To(Equal(testValue))
		})
	})
})
