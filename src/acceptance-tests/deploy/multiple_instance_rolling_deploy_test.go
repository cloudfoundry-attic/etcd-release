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

var _ = Describe("Multiple instance rolling deploys", func() {
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

	It("persists data throughout the rolling deploy", func() {
		keyVals := make(map[string]string)

		By("setting a persistent value", func() {
			var etcdClientURLs []string

			for _, elem := range manifest.Properties.Etcd.Machines {
				etcdClientURLs = append(etcdClientURLs, "http://"+elem+":4001")
			}

			etcdClient := helpers.NewEtcdClient(etcdClientURLs)

			err := etcdClient.Set(testKey, testValue)
			Expect(err).ToNot(HaveOccurred())
		})

		By("deploying", func() {
			manifest.Properties.Etcd.HeartbeatIntervalInMilliseconds = 51

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())

			yaml, err = client.ResolveManifestVersions(yaml)
			Expect(err).NotTo(HaveOccurred())

			var wg sync.WaitGroup
			done := make(chan struct{})

			keysChan := helpers.SpamEtcd(done, &wg, manifest.Properties.Etcd.Machines)

			err = client.Deploy(yaml)
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

		By("reading the value from etcd", func() {
			for _, machine := range manifest.Properties.Etcd.Machines {
				etcdClient := helpers.NewEtcdClient([]string{fmt.Sprintf("http://%s:4001", machine)})

				value, err := etcdClient.Get(testKey)
				Expect(err).ToNot(HaveOccurred())
				Expect(value).To(Equal(testValue))
			}

			for _, machine := range manifest.Properties.Etcd.Machines {
				etcdClient := helpers.NewEtcdClient([]string{fmt.Sprintf("http://%s:4001", machine)})

				for key, value := range keyVals {
					v, err := etcdClient.Get(key)
					Expect(err).ToNot(HaveOccurred())
					Expect(v).To(Equal(value))
				}
			}
		})
	})
})
