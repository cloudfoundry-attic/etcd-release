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

var _ = Describe("Multiple instance rolling upgrade", func() {
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

	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("persists data throughout the rolling upgrade", func() {
		By("deploying the latest final build of etcd-release", func() {
			enableSSL := true
			releaseNumber, err := helpers.DownloadLatestEtcdRelease(client)
			Expect(err).NotTo(HaveOccurred())

			manifest, err = helpers.DeployEtcdWithInstanceCountAndReleaseVersion(3, client, config, enableSSL, releaseNumber)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

			etcdClient = etcdclient.NewClient(fmt.Sprintf("http://%s:6769", manifest.Jobs[2].Networks[0].StaticIPs[0]))
			spammer = helpers.NewSpammer(etcdClient, 1*time.Second, "multiple-instance-rolling-upgrade")
		})

		By("setting a persistent value", func() {
			err := etcdClient.Set(testKey, testValue)
			Expect(err).NotTo(HaveOccurred())
		})

		By("deploying the latest dev build of etcd-release", func() {
			for i := range manifest.Releases {
				if manifest.Releases[i].Name == "etcd" {
					manifest.Releases[i].Version = helpers.EtcdDevReleaseVersion()
				}
			}

			spammer.Spam()

			err := helpers.ResolveVersionsAndDeploy(manifest, client)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

			err = helpers.VerifyDeploymentRelease(client, manifest.Name, helpers.EtcdDevReleaseVersion())
			Expect(err).NotTo(HaveOccurred())

			spammer.Stop()
		})

		By("reading the values from etcd", func() {
			value, err := etcdClient.Get(testKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(testValue))

			err = spammer.Check()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
