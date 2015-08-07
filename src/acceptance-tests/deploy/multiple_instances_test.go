package deploy_test

import (
	"acceptance-tests/helpers"
	"fmt"

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
			helpers.PropertyOverridesStubPath,
			etcdNameOverrideStub,
		)

		for _, elem := range etcdManifest.Properties.Etcd.Machines {
			etcdClientURLs = append(etcdClientURLs, "http://"+elem+":4001")
		}

		By("deploying")
		Expect(bosh.Command("-n", "deploy").Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(len(etcdManifest.Properties.Etcd.Machines)).To(Equal(3))
	})

	AfterEach(func() {
		By("delete deployment")
		Expect(bosh.Command("-n", "delete", "deployment", etcdDeployment).Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("Multiple node deployment", func() {
		It("succesfully deploys multiple etcd node", func() {
			By("deploying")
			Expect(bosh.Command("-n", "deploy").Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))

			By("setting a value on each machine")
			for index, url := range etcdClientURLs {
				etcdClient := etcd.NewClient([]string{url})

				eatsKey := fmt.Sprintf("eats-key%d", index)
				eatsValue := fmt.Sprintf("eats-value%d", index)

				response, err := etcdClient.Create(eatsKey, eatsValue, 60)
				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())
			}

			By("reading each value from each machine")
			for _, url := range etcdClientURLs {
				etcdClient := etcd.NewClient([]string{url})

				for index, _ := range etcdClientURLs {
					eatsKey := fmt.Sprintf("eats-key%d", index)
					eatsValue := fmt.Sprintf("eats-value%d", index)

					response, err := etcdClient.Get(eatsKey, false, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(response.Node.Value).To(Equal(eatsValue))
				}
			}
		})
	})
})
