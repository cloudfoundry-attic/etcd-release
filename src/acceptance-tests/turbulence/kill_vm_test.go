package turbulence_test

import (
	etcdclient "acceptance-tests/testing/etcd"
	"acceptance-tests/testing/helpers"
	"fmt"
	"math/rand"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/etcd"
	"github.com/pivotal-cf-experimental/destiny/turbulence"

	turbulenceclient "github.com/pivotal-cf-experimental/bosh-test/turbulence"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = PDescribe("KillVm", func() {
	KillVMTest := func(enableSSL bool) {
		var (
			turbulenceManifest turbulence.Manifest
			turbulenceClient   turbulenceclient.Client

			etcdManifest etcd.Manifest
			etcdClient   etcdclient.Client

			testKey1   string
			testValue1 string

			testKey2   string
			testValue2 string
		)

		BeforeEach(func() {
			var err error
			turbulenceManifest, err = helpers.DeployTurbulence(boshClient, config)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return boshClient.DeploymentVMs(turbulenceManifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetTurbulenceVMsFromManifest(turbulenceManifest)))

			turbulenceClient = helpers.NewTurbulenceClient(turbulenceManifest)

			guid, err := helpers.NewGUID()
			Expect(err).NotTo(HaveOccurred())

			testKey1 = "etcd-key-1-" + guid
			testValue1 = "etcd-value-1-" + guid

			testKey2 = "etcd-key-2-" + guid
			testValue2 = "etcd-value-2-" + guid

			etcdManifest, err = helpers.DeployEtcdWithInstanceCount(3, boshClient, config, enableSSL)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return boshClient.DeploymentVMs(etcdManifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(etcdManifest)))

			etcdClient = etcdclient.NewClient(fmt.Sprintf("http://%s:6769", etcdManifest.Jobs[2].Networks[0].StaticIPs[0]))
		})

		AfterEach(func() {
			By("deleting the deployment", func() {
				if !CurrentGinkgoTestDescription().Failed {
					err := boshClient.DeleteDeployment(etcdManifest.Name)
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})

		Context("when a etcd node is killed", func() {
			It("is still able to function on healthy vms and recover", func() {
				By("setting a persistent value", func() {
					err := etcdClient.Set(testKey1, testValue1)
					Expect(err).ToNot(HaveOccurred())
				})

				By("killing indices", func() {
					err := turbulenceClient.KillIndices(etcdManifest.Name, "etcd_z1", []int{rand.Intn(3)})
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

					err = boshClient.ScanAndFix(yaml)
					Expect(err).NotTo(HaveOccurred())

					Eventually(func() ([]bosh.VM, error) {
						return boshClient.DeploymentVMs(etcdManifest.Name)
					}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(etcdManifest)))
				})

				By("reading each value from the resurrected VM", func() {
					value, err := etcdClient.Get(testKey1)
					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(Equal(testValue1))

					value, err = etcdClient.Get(testKey2)
					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(Equal(testValue2))
				})
			})
		})
	}

	Context("without TLS", func() {
		KillVMTest(false)
	})

	Context("with TLS", func() {
		KillVMTest(true)
	})
})
