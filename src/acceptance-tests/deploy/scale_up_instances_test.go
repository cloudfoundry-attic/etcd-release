package deploy_test

import (
	"acceptance-tests/testing/helpers"
	"fmt"
	"sync"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Scaling up instances", func() {
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

		manifest, err = helpers.DeployEtcdWithInstanceCount(1, client, config)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return client.DeploymentVMs(manifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{"running"},
		}))

	})

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("scales from 1 to 3 nodes", func() {
		var keyVals map[string]string

		By("setting a persistent value", func() {
			etcdClient := helpers.NewEtcdClient([]string{
				fmt.Sprintf("http://%s:4001", manifest.Properties.Etcd.Machines[0]),
			})

			err := etcdClient.Set(testKey, testValue)
			Expect(err).ToNot(HaveOccurred())
		})

		By("scaling up to 3 nodes", func() {
			manifest.Jobs[0], manifest.Properties = destiny.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, 3)

			members := manifest.EtcdMembers()
			Expect(members).To(HaveLen(3))

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			var wg sync.WaitGroup
			done := make(chan struct{})

			keysChan := helpers.SpamEtcd(done, &wg, manifest.Properties.Etcd.Machines)

			_, err = client.Deploy(yaml)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf([]bosh.VM{
				{"running"},
				{"running"},
				{"running"},
			}))

			close(done)
			wg.Wait()
			keyVals = <-keysChan

			if err, ok := keyVals["error"]; ok {
				Fail(err)
			}

		})

		By("reading the value from each etcd node in the cluster", func() {
			for _, machine := range manifest.Properties.Etcd.Machines {
				etcdClient := helpers.NewEtcdClient([]string{fmt.Sprintf("http://%s:4001", machine)})

				value, err := etcdClient.Get(testKey)
				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal(testValue))
			}

			for _, machine := range manifest.Properties.Etcd.Machines {
				etcdClient := helpers.NewEtcdClient([]string{fmt.Sprintf("http://%s:4001", machine)})

				for k, v := range keyVals {
					value, err := etcdClient.Get(k)
					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(Equal(v))
				}
			}
		})

	})
})
