package deploy_test

import (
	"fmt"
	"time"

	etcdclient "github.com/cloudfoundry-incubator/etcd-release/src/acceptance-tests/testing/etcd"
	"github.com/cloudfoundry-incubator/etcd-release/src/acceptance-tests/testing/helpers"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/ops"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Migrate instance name change", func() {
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

		manifest, err = helpers.DeployEtcdWithInstanceCount("migrate-instance-name-change", 3, false, boshClient)
		Expect(err).NotTo(HaveOccurred())

		manifestName, err = ops.ManifestName(manifest)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return helpers.DeploymentVMs(boshClient, manifestName)
		}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

		testConsumerIPs, err := helpers.GetVMIPs(boshClient, manifestName, "testconsumer")
		Expect(err).NotTo(HaveOccurred())

		etcdClient = etcdclient.NewClient(fmt.Sprintf("http://%s:6769", testConsumerIPs[0]))

		spammer = helpers.NewSpammer(etcdClient, 1*time.Second, "migrate-instance-name-change")
	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := boshClient.DeleteDeployment(manifestName)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("migrates an etcd cluster sucessfully when instance name is changed", func() {
		By("setting a persistent value", func() {
			err := etcdClient.Set(testKey, testValue)
			Expect(err).ToNot(HaveOccurred())
		})

		By("deploying with a new name", func() {
			var err error
			manifest, err = ops.ApplyOps(manifest, []ops.Op{
				{
					Type:  "replace",
					Path:  "/instance_groups/name=etcd/name",
					Value: "new_etcd",
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=new_etcd/migrated_from?/-",
					Value: map[string]string{
						"name": "etcd",
					},
				},
				{
					Type:  "replace",
					Path:  "/instance_groups/name=new_etcd/cluster?/name=etcd/name",
					Value: "new_etcd",
				},
			})
			Expect(err).ToNot(HaveOccurred())

			spammer.Spam()

			_, err = boshClient.Deploy([]byte(manifest))
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return helpers.DeploymentVMs(boshClient, manifestName)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))

			spammer.Stop()
		})

		By("getting a persistent value to verify no data loss during a roll", func() {
			value, err := etcdClient.Get(testKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(testValue))
		})

		By("checking the spammer to verify no downtime", func() {
			spammer.Check()
			read, write := spammer.FailPercentages()

			Expect(read).To(BeNumerically("<=", 4))
			Expect(write).To(BeNumerically("<=", 4))
		})
	})
})
