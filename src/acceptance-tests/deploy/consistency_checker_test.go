package deploy_test

import (
	"acceptance-tests/testing/helpers"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/etcd"
)

var _ = Describe("consistency checker", func() {
	ConsistencyCheckerTest := func(enableSSL bool) {
		It("checks if etcd consistency checker reports failures to bosh during split brain", func() {
			var (
				etcdManifest        etcd.Manifest
				partitionedJobIndex int
				partitionedJobIP    string
				otherJobIPs         []string
			)

			By("deploying etcd cluster", func() {
				configCopy := config
				configCopy.IPTablesAgent = true
				Expect(config.IPTablesAgent).To(BeFalse())

				var err error
				etcdManifest, err = helpers.DeployEtcdWithInstanceCount(3, client, configCopy, enableSSL)
				Expect(err).NotTo(HaveOccurred())
			})

			By("checking if etcd consistency check reports no split brain", func() {
				Eventually(func() ([]bosh.VM, error) {
					return helpers.DeploymentVMs(client, etcdManifest.Name)
				}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(etcdManifest)))
			})

			By("blocking all network traffic on a random etcd node", func() {
				partitionedJobIndex = rand.Intn(3)

				for _, job := range etcdManifest.Jobs {
					if job.Name == "etcd_z1" {
						Expect(job.Networks).To(HaveLen(1))
						Expect(job.Networks[0].StaticIPs).To(HaveLen(3))
						for i, ip := range job.Networks[0].StaticIPs {
							if i == partitionedJobIndex {
								partitionedJobIP = ip
							} else {
								otherJobIPs = append(otherJobIPs, ip)
							}
						}
					}
				}

				err := blockEtcdTraffic(partitionedJobIP, otherJobIPs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("restarting the partitioned node", func() {
				err := client.Restart(etcdManifest.Name, "etcd_z1", partitionedJobIndex)
				Expect(err).NotTo(HaveOccurred())
			})

			By("checking that the partitioned node has elected itself as leader", func() {
				if enableSSL {
					//TODO populate cert, key, ca cert from manifest props
					//etcdURL := fmt.Sprintf("https://%s:4001", partitionedJobIP)
					//goEtcdClient, err = goetcd.NewTLSClient(k.etcdURLs, k.clientCert, k.clientKey, k.caCert)
					//if err != nil {
					//http.Error(w, err.Error(), http.StatusInternalServerError)
					//return
				} else {
					resp, err := http.Get(fmt.Sprintf("http://%s:4001/v2/stats/self", partitionedJobIP))
					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(http.StatusOK))

					type selfResponse struct {
						ID string `json:"id"`
					}

					var selfResp selfResponse
					err = json.NewDecoder(resp.Body).Decode(&selfResp)
					Expect(err).NotTo(HaveOccurred())

					type leaderResponse struct {
						Leader    string                 `json:leader`
						Followers map[string]interface{} `json:followers`
					}

					Eventually(func() leaderResponse {
						resp, err := http.Get(fmt.Sprintf("http://%s:4001/v2/stats/leader", partitionedJobIP))
						Expect(err).NotTo(HaveOccurred())
						Expect(resp.StatusCode).To(Equal(http.StatusOK))

						var leaderResp leaderResponse
						err = json.NewDecoder(resp.Body).Decode(&leaderResp)
						Expect(err).NotTo(HaveOccurred())

						return leaderResp
					}, "2s", "10s").Should(Equal(leaderResponse{
						Leader:    selfResp.ID,
						Followers: map[string]interface{}{},
					}))
				}
			})

			By("removing the blockage of traffic on the partitioned node", func() {
				err := unblockEtcdTraffic(partitionedJobIP, otherJobIPs)
				Expect(err).NotTo(HaveOccurred())
			})

			By("checking if etcd consistency check reports a split brain", func() {
				vms := []bosh.VM{}
				for _, vm := range helpers.GetVMsFromManifest(etcdManifest) {
					if vm.JobName == "etcd_z1" {
						vm.State = "failing"
					}
					vms = append(vms, vm)
				}

				Eventually(func() ([]bosh.VM, error) {
					return helpers.DeploymentVMs(client, etcdManifest.Name)
				}, "1m", "10s").Should(ConsistOf(vms))
			})

			By("deleting the deployment", func() {
				err := client.DeleteDeployment(etcdManifest.Name)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	}

	FContext("without TLS", func() {
		ConsistencyCheckerTest(false)
	})

	Context("with TLS", func() {
		ConsistencyCheckerTest(true)
	})
})

func blockEtcdTraffic(machineIP string, etcdJobIPs []string) error {
	ports := []int{4001, 7001}

	for _, port := range ports {
		for _, etcdJobIP := range etcdJobIPs {
			req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s:5678/drop?addr=%s&port=%d", machineIP, etcdJobIP, port), strings.NewReader(""))
			if err != nil {
				return err
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				respBody, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					respBody = []byte("could not read body")
				}

				return fmt.Errorf("unexpected status: %d, error: %s", resp.StatusCode, string(respBody))
			}
		}
	}
	return nil
}

func unblockEtcdTraffic(machineIP string, etcdJobIPs []string) error {
	ports := []int{4001, 7001}

	for _, port := range ports {
		for _, etcdJobIP := range etcdJobIPs {
			req, err := http.NewRequest("DELETE", fmt.Sprintf("http://%s:5678/drop?addr=%s&port=%d", machineIP, etcdJobIP, port), strings.NewReader(""))
			if err != nil {
				return err
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				respBody, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					respBody = []byte("could not read body")
				}

				return fmt.Errorf("unexpected status: %d, error: %s", resp.StatusCode, string(respBody))
			}
		}
	}
	return nil
}
