package deploy_test

import (
	"acceptance-tests/testing/helpers"
	"fmt"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scaling down instances", func() {
	var (
		manifest destiny.Manifest

		testKey   string
		testValue string
	)

	BeforeEach(func() {
		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		testKey = "etcd-key-" + guid
		testValue = "etcd-value-" + guid

		manifest, err = helpers.DeployEtcdWithInstanceCount(3, client, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
			{"running"},
			{"running"},
		}))
	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("scales from 3 to 1 nodes", func() {
		By("setting a persistent value", func() {
			etcdClient := helpers.NewEtcdClient([]string{
				fmt.Sprintf("http://%s:4001", manifest.Properties.Etcd.Machines[1]),
			})

			err := etcdClient.Set(testKey, testValue)
			Expect(err).ToNot(HaveOccurred())
		})

		By("scaling down to 1 node", func() {
			manifest.Jobs[0], manifest.Properties = destiny.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 1)

			members := manifest.EtcdMembers()
			Expect(members).To(HaveLen(1))

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			err = client.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
			}))
		})

		By("reading the value from etcd", func() {
			etcdClient := helpers.NewEtcdClient([]string{
				fmt.Sprintf("http://%s:4001", manifest.Properties.Etcd.Machines[0]),
			})

			value, err := etcdClient.Get(testKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(value).To(Equal(testValue))
		})
	})
})
