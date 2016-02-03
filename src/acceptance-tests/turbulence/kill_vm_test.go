package turbulence_test

import (
	"acceptance-tests/testing/bosh"
	"acceptance-tests/testing/destiny"
	"acceptance-tests/testing/etcd"
	"acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("KillVm", func() {
	var (
		etcdManifest destiny.Manifest
		etcdClient   etcd.Client

		testKey        string
		testValue      string
		etcdClientURLs []string
	)

	BeforeEach(func() {
		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		testKey = "etcd-key-" + guid
		testValue = "etcd-value-" + guid

		etcdManifest, err = helpers.DeployEtcdWithInstanceCount(3, client, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(etcdManifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
			{"running"},
			{"running"},
		}))
	})

	AfterEach(func() {
		By("fixing the deployment", func() {
			yaml, err := etcdManifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			err = client.ScanAndFix(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(etcdManifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
			}))
		})

		By("deleting the deployment", func() {
			err := client.DeleteDeployment(etcdManifest.Name)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when a etcd node is killed", func() {
		It("is still able to function on healthy vms", func() {
			By("killing indices", func() {
				err := turbulenceClient.KillIndices(etcdManifest.Name, "etcd_z1", []int{0})
				Expect(err).ToNot(HaveOccurred())
			})

			By("creating an etcd client connection", func() {
				etcdClientURLs = append(etcdClientURLs, "http://"+etcdManifest.Properties.Etcd.Machines[1]+":4001")
				etcdClientURLs = append(etcdClientURLs, "http://"+etcdManifest.Properties.Etcd.Machines[2]+":4001")

				etcdClient = NewEtcdClient(etcdClientURLs)
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
})
