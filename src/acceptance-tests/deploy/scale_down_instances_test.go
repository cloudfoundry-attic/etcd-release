package deploy_test

import (
	"acceptance-tests/helpers"

	"github.com/coreos/go-etcd/etcd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Multiple Instances", func() {
	var (
		etcdManifest   = new(helpers.Manifest)
		etcdClientURLs []string
	)

	BeforeEach(func() {
		bosh.GenerateAndSetDeploymentManifest(
			etcdManifest,
			etcdManifestGeneration,
			directorUUIDStub,
			helpers.InstanceCount3NodesStubPath,
			helpers.PersistentDiskStubPath,
			config.IAASSettingsEtcdStubPath,
			etcdNameOverrideStub,
		)

		By("deploying")
		Expect(bosh.Command("-n", "deploy").Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(len(etcdManifest.Properties.Etcd.Machines)).To(Equal(3))
	})

	AfterEach(func() {
		By("delete deployment")
		Expect(bosh.Command("-n", "delete", "deployment", etcdDeployment).Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("scaling from 3 node to 1", func() {
		It("succesfully scales to multiple etcd nodes", func() {
			bosh.GenerateAndSetDeploymentManifest(
				etcdManifest,
				etcdManifestGeneration,
				directorUUIDStub,
				helpers.InstanceCount1NodeStubPath,
				helpers.PersistentDiskStubPath,
				config.IAASSettingsEtcdStubPath,
				etcdNameOverrideStub,
			)

			for _, elem := range etcdManifest.Properties.Etcd.Machines {
				etcdClientURLs = append(etcdClientURLs, "http://"+elem+":4001")
			}

			By("deploying")
			Expect(bosh.Command("-n", "deploy").Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))
			Expect(len(etcdManifest.Properties.Etcd.Machines)).To(Equal(1))

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
})
