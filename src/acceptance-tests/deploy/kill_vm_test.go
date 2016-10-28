package deploy_test

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

var _ = Describe("KillVm", func() {
	KillVMTest := func(enableSSL bool) {
		var (
			etcdManifest etcd.Manifest
			etcdClient   etcdclient.Client

			turbulenceManifest turbulence.Manifest
			turbulenceClient   turbulenceclient.Client

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

			By("deploying turbulence", func() {
				var err error
				turbulenceManifest, err = helpers.DeployTurbulence("kill_vm", client, config)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return helpers.DeploymentVMs(client, turbulenceManifest.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetTurbulenceVMsFromManifest(turbulenceManifest)))

				turbulenceClient = helpers.NewTurbulenceClient(turbulenceManifest)
			})

			By("deploying 3 node etcd", func() {
				etcdManifest, err = helpers.DeployEtcdWithInstanceCount("kill_vm", 3, client, config, enableSSL)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return helpers.DeploymentVMs(client, etcdManifest.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(etcdManifest)))

				testConsumerIndex, err := helpers.FindJobIndexByName(etcdManifest, "testconsumer_z1")
				Expect(err).NotTo(HaveOccurred())
				etcdClient = etcdclient.NewClient(fmt.Sprintf("http://%s:6769", etcdManifest.Jobs[testConsumerIndex].Networks[0].StaticIPs[0]))
			})
		})

		AfterEach(func() {
			By("deleting the deployment", func() {
				if !CurrentGinkgoTestDescription().Failed {
					yaml, err := etcdManifest.ToYAML()
					Expect(err).NotTo(HaveOccurred())

					Eventually(func() ([]string, error) {
						return lockedDeployments(client)
					}, "10m", "1m").ShouldNot(ContainElement(etcdManifest.Name))

					err = client.ScanAndFixAll(yaml)
					Expect(err).NotTo(HaveOccurred())

					err = client.DeleteDeployment(etcdManifest.Name)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			By("deleting turbulence", func() {
				err := client.DeleteDeployment(turbulenceManifest.Name)
				Expect(err).NotTo(HaveOccurred())
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
					Eventually(func() (string, error) {
						return etcdClient.Get(testKey1)
					}, "10s", "1s").Should(Equal(testValue1))
				})

				By("setting a new persistent value", func() {
					Eventually(func() error {
						return etcdClient.Set(testKey2, testValue2)
					}, "10s", "1s").Should(Succeed())
				})

				By("fixing the deployment", func() {
					yaml, err := etcdManifest.ToYAML()
					Expect(err).NotTo(HaveOccurred())

					Eventually(func() ([]string, error) {
						return lockedDeployments(client)
					}, "10m", "1m").ShouldNot(ContainElement(etcdManifest.Name))

					err = client.ScanAndFixAll(yaml)
					Expect(err).NotTo(HaveOccurred())

					Eventually(func() ([]bosh.VM, error) {
						return helpers.DeploymentVMs(client, etcdManifest.Name)
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
