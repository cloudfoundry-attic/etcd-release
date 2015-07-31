package deploy_test

import (
	"acceptance-tests/helpers"

	"github.com/coreos/go-etcd/etcd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("SingleInstance", func() {
	var (
		etcdClientURLs []string
	)

	BeforeEach(func() {

		etcdClientURLs = bosh.GenerateAndSetDeploymentManifest(
			directorUUIDStub.Name(),
			helpers.InstanceCount1NodeStubPath,
			helpers.PersistentDiskStubPath,
			config.IAASSettingsStubPath,
			nameOverridesStub.Name(),
		)

		By("deploying")
		Expect(bosh.Command("-n", "deploy").Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))
	})

	AfterEach(func() {
		By("delete deployment")
		Expect(bosh.Command("-n", "delete", "deployment", etcdName).Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("deploys one etcd node", func() {
		Expect(len(etcdClientURLs)).To(Equal(1))
		for index, value := range etcdClientURLs {
			etcdClient := etcd.NewClient([]string{value})
			eatsKey := "eats-key" + string(index)
			eatsValue := "eats-value" + string(index)

			response, err := etcdClient.Create(eatsKey, eatsValue, 60)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())

			response, err = etcdClient.Get(eatsKey, false, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Node.Value).To(Equal(eatsValue))
		}
	})
})
