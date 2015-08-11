package turbulence_test

import (
	"acceptance-tests/helpers"
	"acceptance-tests/turbulence/client"
	"fmt"
	"time"

	"github.com/coreos/go-etcd/etcd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("KillVm", func() {
	var (
		etcdManifest  = new(helpers.Manifest)
		turbulenceUrl string

		etcdClientURLs []string
		killedEtcdUrls []string
		aliveEtcdUrls  []string
	)

	BeforeEach(func() {
		turbulenceUrl = "https://turbulence:" + turbulenceManifest.Properties.TurbulenceApi.Password + "@" + turbulenceManifest.Jobs[0].Networks[0].StaticIps[0] + ":8080"

		By("generating etcd manifest")
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

		By("deploying")
		Expect(bosh.Command("-n", "deploy").Wait(config.DefaultTimeout)).To(Exit(0))
		Expect(len(etcdManifest.Properties.Etcd.Machines)).To(Equal(3))

		for _, elem := range etcdManifest.Properties.Etcd.Machines {
			etcdClientURLs = append(etcdClientURLs, "http://"+elem+":4001")
		}

		aliveEtcdUrls = []string{
			"http://" + etcdManifest.Jobs[1].Networks[0].StaticIps[0] + ":4001",
			"http://" + etcdManifest.Jobs[1].Networks[0].StaticIps[1] + ":4001",
		}

		killedEtcdUrls = []string{
			"http://" + etcdManifest.Jobs[0].Networks[0].StaticIps[0] + ":4001",
		}
	})

	AfterEach(func() {
		By("Fixing the release")
		Expect(bosh.Command("cck", "--auto").Wait(config.DefaultTimeout)).To(Exit(0))

		By("delete deployment")
		Expect(bosh.Command("-n", "delete", "deployment", etcdDeployment).Wait(config.DefaultTimeout)).To(Exit(0))
	})

	Context("When an etcd node is killed", func() {
		BeforeEach(func() {
			turbulenceClient := client.NewClient(turbulenceUrl)

			err := turbulenceClient.KillIndices(etcdDeployment, "etcd_z1", []int{0})
			Expect(err).ToNot(HaveOccurred())
		})

		It("Is still able to function on healthy vms", func() {
			for index, value := range aliveEtcdUrls {
				etcdClient := etcd.NewClient([]string{value})

				eatsKey := fmt.Sprintf("eats-key%d", index)
				eatsValue := fmt.Sprintf("eats-value%d", index)

				response, err := etcdClient.Create(eatsKey, eatsValue, 60)
				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())
			}

			time.Sleep(100 * time.Millisecond)
			for _, url := range aliveEtcdUrls {
				etcdClient := etcd.NewClient([]string{url})

				for index, _ := range aliveEtcdUrls {
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
