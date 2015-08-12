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
		etcdManifest   = new(helpers.Manifest)
		etcdClientURLs []string
	)

	BeforeEach(func() {
		By("generating etcd manifest")
		bosh.GenerateAndSetDeploymentManifest(
			etcdManifest,
			etcdManifestGeneration,
			directorUUIDStub,
			helpers.InstanceCount1NodeStubPath,
			helpers.PersistentDiskStubPath,
			config.IAASSettingsEtcdStubPath,
			helpers.PropertyOverridesStubPath,
			etcdNameOverrideStub,
		)

		for _, elem := range etcdManifest.Properties.Etcd.Machines {
			etcdClientURLs = append(etcdClientURLs, "http://"+elem+":4001")
		}

		By("deploying")
		Expect(bosh.Command("-n", "deploy")).To(Exit(0))
		Expect(len(etcdManifest.Properties.Etcd.Machines)).To(Equal(1))
	})

	AfterEach(func() {
		By("delete deployment")
		bosh.Command("-n", "delete", "deployment", etcdDeployment)
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
