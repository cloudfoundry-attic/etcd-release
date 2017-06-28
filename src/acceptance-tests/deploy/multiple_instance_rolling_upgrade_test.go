package deploy_test

import (
	"fmt"
	"os"
	"time"

	etcdclient "github.com/cloudfoundry-incubator/etcd-release/src/acceptance-tests/testing/etcd"

	"github.com/cloudfoundry-incubator/etcd-release/src/acceptance-tests/testing/helpers"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/ops"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multiple instance rolling upgrade", func() {
	var (
		manifest     string
		manifestName string

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

		releaseNumber := os.Getenv("LATEST_ETCD_RELEASE_VERSION")

		enableSSL := true
		manifest, err = helpers.DeployEtcdWithInstanceCountAndReleaseVersion("multiple-instance-rolling-upgrade", 3, enableSSL, boshClient, releaseNumber)
		Expect(err).NotTo(HaveOccurred())

		manifestName, err = ops.ManifestName(manifest)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return helpers.DeploymentVMs(boshClient, manifestName)
		}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

		testConsumerIPs, err := helpers.GetVMIPs(boshClient, manifestName, "testconsumer")
		Expect(err).NotTo(HaveOccurred())

		etcdClient = etcdclient.NewClient(fmt.Sprintf("http://%s:6769", testConsumerIPs[0]))

		spammer = helpers.NewSpammer(etcdClient, 1*time.Second, "multiple-instance-rolling-upgrade")
	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := boshClient.DeleteDeployment(manifestName)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("persists data throughout the rolling upgrade", func() {
		By("setting a persistent value", func() {
			err := etcdClient.Set(testKey, testValue)
			Expect(err).NotTo(HaveOccurred())
		})

		By("deploying the latest dev build of etcd-release", func() {
			var err error
			manifest, err = ops.ApplyOp(manifest, ops.Op{
				Type:  "replace",
				Path:  "/releases/name=etcd/version",
				Value: helpers.EtcdDevReleaseVersion(),
			})
			Expect(err).NotTo(HaveOccurred())

			spammer.Spam()

			_, err = boshClient.Deploy([]byte(manifest))
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return helpers.DeploymentVMs(boshClient, manifestName)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

			err = helpers.VerifyDeploymentRelease(boshClient, manifestName, helpers.EtcdDevReleaseVersion())
			Expect(err).NotTo(HaveOccurred())

			spammer.Stop()
		})

		By("reading the values from etcd", func() {
			value, err := etcdClient.Get(testKey)
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal(testValue))

			spammer.Check()
			read, write := spammer.FailPercentages()

			Expect(read).To(BeNumerically("<=", 4))
			Expect(write).To(BeNumerically("<=", 4))
		})
	})
})
