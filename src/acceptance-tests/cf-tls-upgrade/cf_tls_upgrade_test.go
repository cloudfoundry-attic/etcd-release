package cf_tls_upgrade_test

import (
	"acceptance-tests/testing/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
)

var _ = Describe("CF TLS Upgrade Test", func() {
	It("successfully upgrades etcd cluster to use TLS", func() {
		var (
			originalManifest  []byte
			migrationManifest []byte
			err               error
		)

		By("downloading existing manifest", func() {
			originalManifest, err = client.DownloadManifest("cf-warden")
			Expect(err).NotTo(HaveOccurred())
		})

		By("scaling down the non-TLS etcd cluster to 1 node and converting it to a proxy", func() {
			migrationManifest, err = helpers.CreateCFTLSMigrationManifest(originalManifest)
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Deploy(migrationManifest)
			Expect(err).NotTo(HaveOccurred())
		})

		By("checking checking if expected VMs are running", func() {
			expectedVMs, err := helpers.GetVMsFromRawManifest(migrationManifest)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs("cf-warden")
			}, "1m", "10s").Should(ConsistOf(expectedVMs))
		})
	})
})
