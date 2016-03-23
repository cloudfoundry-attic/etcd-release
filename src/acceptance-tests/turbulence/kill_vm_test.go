package turbulence_test

import (
	"acceptance-tests/testing/etcd"
	"acceptance-tests/testing/helpers"
	"fmt"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = PDescribe("KillVm", func() {
	var (
		etcdManifest destiny.Manifest
		etcdClient   etcd.Client

		testKey1   string
		testValue1 string

		testKey2   string
		testValue2 string
	)

	BeforeEach(func() {
		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		testKey1 = "etcd-key-1-" + guid
		testValue1 = "etcd-value-1-" + guid

		testKey2 = "etcd-key-2-" + guid
		testValue2 = "etcd-value-2-" + guid

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
		By("deleting the deployment", func() {
			if !CurrentGinkgoTestDescription().Failed {
				err := client.DeleteDeployment(etcdManifest.Name)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Context("when a etcd node is killed", func() {
		It("is still able to function on healthy vms and recover", func() {
			By("creating an etcd client connection", func() {
				var etcdClientURLs []string

				for _, machine := range etcdManifest.Properties.Etcd.Machines {
					etcdClientURLs = append(etcdClientURLs, fmt.Sprintf("http://%s:4001", machine))
				}

				etcdClient = helpers.NewEtcdClient(etcdClientURLs)
			})

			By("setting a persistent value", func() {
				err := etcdClient.Set(testKey1, testValue1)
				Expect(err).ToNot(HaveOccurred())
			})

			By("killing indices", func() {
				err := turbulenceClient.KillIndices(etcdManifest.Name, "etcd_z1", []int{0})
				Expect(err).ToNot(HaveOccurred())
			})

			By("reading the value from etcd", func() {
				value, err := etcdClient.Get(testKey1)
				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal(testValue1))
			})

			By("setting a new persistent value", func() {
				err := etcdClient.Set(testKey2, testValue2)
				Expect(err).ToNot(HaveOccurred())
			})

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

			By("reading each value from the resurrected VM", func() {
				etcdClient := helpers.NewEtcdClient([]string{
					fmt.Sprintf("http://%s:4001", etcdManifest.Properties.Etcd.Machines[0]),
				})

				value, err := etcdClient.Get(testKey1)
				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal(testValue1))

				value, err = etcdClient.Get(testKey2)
				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal(testValue2))
			})
		})
	})
})
