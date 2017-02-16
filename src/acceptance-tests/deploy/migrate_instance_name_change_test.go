package deploy_test

import (
	"acceptance-tests/testing/helpers"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/etcd"

	etcdclient "acceptance-tests/testing/etcd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Migrate instance name change", func() {
	var (
		manifest   string
		etcdClient etcdclient.Client
		spammer    *helpers.Spammer

		testKey      string
		testValue    string
		manifestName string
	)

	BeforeEach(func() {
		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		testKey = "etcd-key-" + guid
		testValue = "etcd-value-" + guid

		manifest, err = helpers.DeployEtcdV2WithInstanceCount("migrate_instance_name_change", 3, client, config)
		Expect(err).NotTo(HaveOccurred())

		manifestName, err = etcd.ManifestName(manifest)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return helpers.DeploymentVMs(client, manifestName)
		}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifestV2(manifest)))

		vmIPs, err := helpers.GetVMIPs(client, manifestName, "testconsumer")
		Expect(err).NotTo(HaveOccurred())
		etcdClient = etcdclient.NewClient(fmt.Sprintf("http://%s:6769", vmIPs[0]))
		spammer = helpers.NewSpammer(etcdClient, 1*time.Second, "migrate-instance-name-change")
	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifestName)
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
			manifest, err = etcd.ApplyOp(manifest, "replace", "/instance_groups/name=etcd/name", "new_etcd")
			Expect(err).ToNot(HaveOccurred())

			manifest, err = etcd.ApplyOp(manifest, "replace", "/instance_groups/name=new_etcd/migrated_from?", []map[string]string{{"name": "etcd"}})
			Expect(err).ToNot(HaveOccurred())

			manifest, err = etcd.ApplyOp(manifest, "replace", "/properties/etcd/cluster/name=etcd/name", "new_etcd")
			Expect(err).ToNot(HaveOccurred())

			spammer.Spam()

			err = helpers.ResolveVersionsAndDeployV2(manifest, client)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return helpers.DeploymentVMs(client, manifestName)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifestV2(manifest)))

			spammer.Stop()
		})

		By("getting a persistent value", func() {
			value, err := etcdClient.Get(testKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(testValue))
		})

		By("checking the spammer", func() {
			spammerErrs := spammer.Check()

			var errorSet helpers.ErrorSet

			switch spammerErrs.(type) {
			case helpers.ErrorSet:
				errorSet = spammerErrs.(helpers.ErrorSet)
			case nil:
				return
			default:
				Fail(spammerErrs.Error())
			}

			tcpErrCount := 0
			unexpectedErrCount := 0
			testConsumerConnectionResetErrorCount := 0
			otherErrors := helpers.ErrorSet{}

			for err, occurrences := range errorSet {
				switch {
				// This happens when the etcd leader is killed and a request is issued while an election is happening
				case strings.Contains(err, "Unexpected HTTP status code"):
					unexpectedErrCount += occurrences
				// This happens when the consul_agent gets rolled when a request is sent to the testconsumer
				case strings.Contains(err, "dial tcp: lookup etcd.service.cf.internal on"):
					tcpErrCount += occurrences
				// This happens when a connection is severed by the etcd server
				case strings.Contains(err, "EOF"):
					testConsumerConnectionResetErrorCount += occurrences
				default:
					otherErrors.Add(errors.New(err))
				}
			}

			Expect(otherErrors).To(HaveLen(0))
			Expect(unexpectedErrCount).To(BeNumerically("<=", 3))
			Expect(tcpErrCount).To(BeNumerically("<=", 1))
			Expect(testConsumerConnectionResetErrorCount).To(BeNumerically("<=", 1))
		})
	})
})
