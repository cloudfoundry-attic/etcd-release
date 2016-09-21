package deploy_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"acceptance-tests/testing/helpers"

	etcdclient "acceptance-tests/testing/etcd"

	turbulenceclient "github.com/pivotal-cf-experimental/bosh-test/turbulence"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/etcd"
	"github.com/pivotal-cf-experimental/destiny/turbulence"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("quorum loss", func() {
	QuorumLossTest := func(enableSSL bool) {
		var (
			turbulenceManifest turbulence.Manifest
			turbulenceClient   turbulenceclient.Client

			etcdManifest etcd.Manifest
			etcdClient   etcdclient.Client
		)

		BeforeEach(func() {
			var err error
			turbulenceManifest, err = helpers.DeployTurbulence(client, config)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return helpers.DeploymentVMs(client, turbulenceManifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetTurbulenceVMsFromManifest(turbulenceManifest)))

			turbulenceClient = helpers.NewTurbulenceClient(turbulenceManifest)

			etcdManifest, err = helpers.DeployEtcdWithInstanceCount(5, client, config, enableSSL)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return helpers.DeploymentVMs(client, etcdManifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(etcdManifest)))

			etcdClient = etcdclient.NewClient(fmt.Sprintf("http://%s:6769", etcdManifest.Jobs[2].Networks[0].StaticIPs[0]))
		})

		AfterEach(func() {
			By("deleting the deployment", func() {
				if !CurrentGinkgoTestDescription().Failed {
					for i := 0; i < 5; i++ {
						err := client.SetVMResurrection(etcdManifest.Name, "etcd_z1", i, true)
						Expect(err).NotTo(HaveOccurred())
					}

					yaml, err := etcdManifest.ToYAML()
					Expect(err).NotTo(HaveOccurred())

					Eventually(func() error {
						return client.ScanAndFixAll(yaml)
					}, "5m", "1m").ShouldNot(HaveOccurred())

					Eventually(func() ([]bosh.VM, error) {
						return helpers.DeploymentVMs(client, etcdManifest.Name)
					}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(etcdManifest)))

					err = client.DeleteDeployment(etcdManifest.Name)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			By("deleting the turbulence deployment", func() {
				if !CurrentGinkgoTestDescription().Failed {
					err := client.DeleteDeployment(turbulenceManifest.Name)
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})

		Context("when a etcd node is killed", func() {
			It("is still able to function on healthy vms", func() {
				By("setting and getting a value", func() {
					guid, err := helpers.NewGUID()
					Expect(err).NotTo(HaveOccurred())
					testKey := "etcd-key-" + guid
					testValue := "etcd-value-" + guid

					err = etcdClient.Set(testKey, testValue)
					Expect(err).NotTo(HaveOccurred())

					value, err := etcdClient.Get(testKey)
					Expect(err).NotTo(HaveOccurred())
					Expect(value).To(Equal(testValue))
				})

				By("killing indices", func() {
					for i := 0; i < 5; i++ {
						err := client.SetVMResurrection(etcdManifest.Name, "etcd_z1", i, false)
						Expect(err).NotTo(HaveOccurred())
					}

					leader, err := jobIndexOfLeader(fmt.Sprintf("http://%s:6769", etcdManifest.Jobs[2].Networks[0].StaticIPs[0]))
					Expect(err).NotTo(HaveOccurred())

					rand.Seed(time.Now().Unix())
					startingIndex := rand.Intn(3)
					instances := []int{startingIndex, startingIndex + 1, startingIndex + 2}

					if leader < startingIndex || leader > startingIndex+2 {
						instances[0] = leader
					}

					jobIndexToResurrect := startingIndex + 1

					err = turbulenceClient.KillIndices(etcdManifest.Name, "etcd_z1", instances)
					Expect(err).NotTo(HaveOccurred())

					err = client.SetVMResurrection(etcdManifest.Name, "etcd_z1", jobIndexToResurrect, true)
					Expect(err).NotTo(HaveOccurred())

					Eventually(func() error {
						return client.ScanAndFix(etcdManifest.Name, "etcd_z1", []int{jobIndexToResurrect})
					}, "5m", "1m").ShouldNot(HaveOccurred())

					Eventually(func() ([]bosh.VM, error) {
						return helpers.DeploymentVMs(client, etcdManifest.Name)
					}, "5m", "1m").Should(ContainElement(bosh.VM{JobName: "etcd_z1", Index: jobIndexToResurrect, State: "running"}))
				})

				By("setting and getting a new value", func() {
					guid, err := helpers.NewGUID()
					Expect(err).NotTo(HaveOccurred())
					testKey := "etcd-key-" + guid
					testValue := "etcd-value-" + guid

					err = etcdClient.Set(testKey, testValue)
					Expect(err).NotTo(HaveOccurred())

					value, err := etcdClient.Get(testKey)
					Expect(err).NotTo(HaveOccurred())
					Expect(value).To(Equal(testValue))
				})
			})
		})
	}

	Context("without TLS", func() {
		QuorumLossTest(false)
	})

	Context("with TLS", func() {
		QuorumLossTest(true)
	})
})

func jobIndexOfLeader(testconsumerURL string) (int, error) {
	resp, err := http.Get(fmt.Sprintf("%s/leader_name", testconsumerURL))
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("%s", string(buf))
	}

	leaderNameParts := strings.Split(string(buf), "-")

	leader, err := strconv.Atoi(leaderNameParts[len(leaderNameParts)-1])
	if err != nil {
		return -1, err
	}

	return leader, nil
}
