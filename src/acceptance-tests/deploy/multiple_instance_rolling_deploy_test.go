package deploy_test

import (
	"fmt"
	"time"

	etcdclient "acceptance-tests/testing/etcd"

	"acceptance-tests/testing/helpers"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/etcd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multiple instance rolling deploys", func() {
	MultipleInstanceRollingDeploy := func(enableSSL bool) {
		var (
			manifest   etcd.Manifest
			etcdClient etcdclient.Client
			spammer    *helpers.Spammer

			testKey   string
			testValue string
		)

		BeforeEach(func() {
			guid, err := helpers.NewGUID()
			Expect(err).NotTo(HaveOccurred())

			testKey = "etcd-key-" + guid
			testValue = "etcd-value-" + guid

			manifest, err = helpers.DeployEtcdWithInstanceCount(3, client, config, enableSSL)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

			etcdClient = etcdclient.NewClient(fmt.Sprintf("http://%s:6769", manifest.Jobs[2].Networks[0].StaticIPs[0]))
			spammer = helpers.NewSpammer(etcdClient, 1*time.Second, "multi-instance-rolling-deploy")
		})

		AfterEach(func() {
			if !CurrentGinkgoTestDescription().Failed {
				err := client.DeleteDeployment(manifest.Name)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("persists data throughout the rolling deploy", func() {
			By("setting a persistent value", func() {
				err := etcdClient.Set(testKey, testValue)
				Expect(err).ToNot(HaveOccurred())
			})

			By("deploying", func() {
				manifest.Properties.Etcd.HeartbeatIntervalInMilliseconds = 51

				yaml, err := manifest.ToYAML()
				Expect(err).NotTo(HaveOccurred())

				yaml, err = client.ResolveManifestVersions(yaml)
				Expect(err).NotTo(HaveOccurred())

				spammer.Spam()

				_, err = client.Deploy(yaml)
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() ([]bosh.VM, error) {
					return client.DeploymentVMs(manifest.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

				spammer.Stop()
			})

			By("reading the value from etcd", func() {
				value, err := etcdClient.Get(testKey)
				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal(testValue))

				err = spammer.Check()
				Expect(err).ToNot(HaveOccurred())
			})
		})
	}

	Context("without TLS", func() {
		MultipleInstanceRollingDeploy(false)
	})

	Context("with TLS", func() {
		MultipleInstanceRollingDeploy(true)
	})
})
