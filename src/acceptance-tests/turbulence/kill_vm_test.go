package turbulence_test

import (
	"acceptance-tests/helpers"
	"acceptance-tests/turbulence/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("KillVm", func() {
	var (
		etcdClientURLs []string
	)

	BeforeEach(func() {
		etcdClientURLs = bosh.GenerateAndSetDeploymentManifest(
			directorUUIDStub.Name(),
			helpers.InstanceCount3NodesStubPath,
			helpers.PersistentDiskStubPath,
			config.IAASSettingsEtcdStubPath,
			etcdNameOverrideStub.Name(),
		)

		By("deploying")
		Expect(bosh.Command("-n", "deploy").Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))

		Expect(len(etcdClientURLs)).To(Equal(3))
	})

	AfterEach(func() {
		By("delete deployment")
		Expect(bosh.Command("-n", "delete", "deployment", etcdName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("kills one etcd node", func() {
		turbulenceClient := client.NewClient(turbulencUrl)

		err := turbulenceClient.KillIndices(etcdName, "etcd_z2", []int{1})
		Expect(err).ToNot(HaveOccurred())
	})
})
