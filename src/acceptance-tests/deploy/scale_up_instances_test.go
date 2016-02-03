package deploy_test

import (
	"acceptance-tests/testing/bosh"
	"acceptance-tests/testing/destiny"
	"acceptance-tests/testing/etcd"
	"acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scaling up instances", func() {
	var (
		manifest   destiny.Manifest
		etcdClient etcd.Client

		testKey        string
		testValue      string
		etcdClientURLs []string
	)

	BeforeEach(func() {
		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		testKey = "etcd-key-" + guid
		testValue = "etcd-value-" + guid

		manifest, err = helpers.DeployEtcdWithInstanceCount(1, client, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
		}))

	})

	AfterEach(func() {
		err := client.DeleteDeployment(manifest.Name)
		Expect(err).NotTo(HaveOccurred())
	})

	It("scales from 1 to 3 nodes", func() {
		By("scaling up to 3 nodes", func() {
			manifest.Jobs[0], manifest.Properties = destiny.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 3)

			members := manifest.EtcdMembers()
			Expect(members).To(HaveLen(3))

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			err = client.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
			}))
		})

		By("instantiating a etcd client connection", func() {
			for _, elem := range manifest.Properties.Etcd.Machines {
				etcdClientURLs = append(etcdClientURLs, "http://"+elem+":4001")
			}

			etcdClient = helpers.NewEtcdClient(etcdClientURLs)
		})

		By("setting a persistent value", func() {
			err := etcdClient.Set(testKey, testValue)
			Expect(err).ToNot(HaveOccurred())
		})

		By("reading the value from etcd", func() {
			value, err := etcdClient.Get(testKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(testValue))
		})
	})
})
