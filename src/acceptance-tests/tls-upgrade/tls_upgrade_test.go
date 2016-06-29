package tls_upgrade_test

import (
	etcdclient "acceptance-tests/testing/etcd"
	"acceptance-tests/testing/helpers"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/core"
	"github.com/pivotal-cf-experimental/destiny/etcd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	PUT_ERROR_COUNT_THRESHOLD                  = 5
	TEST_CONSUMER_CONNECTION_RESET_ERROR_COUNT = 3
)

var _ = Describe("TLS Upgrade", func() {
	var (
		manifest   etcd.Manifest
		etcdClient etcdclient.Client
		spammer    *helpers.Spammer
	)

	var findJob = func(manifest etcd.Manifest, jobName string) (core.Job, error) {
		for _, job := range manifest.Jobs {
			if job.Name == jobName {
				return job, nil
			}
		}
		return core.Job{}, errors.New("job not found")
	}

	var getEtcdProxyState = func() (string, error) {
		vms, err := client.DeploymentVMs(manifest.Name)
		if err != nil {
			return "", err
		}
		for _, vm := range vms {
			if vm.JobName == "etcd_z1" && vm.Index == 0 {
				return vm.State, nil
			}
		}

		return "not found", err
	}

	AfterEach(func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := client.DeleteDeployment(manifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("keeps writing to an etcd cluster without interruption", func() {
		By("deploy non tls etcd", func() {
			var err error
			manifest, err = helpers.DeployEtcdWithInstanceCount(3, client, config, false)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
		})

		By("spamming the cluster", func() {
			testConsumerJob, err := findJob(manifest, "testconsumer_z1")
			Expect(err).NotTo(HaveOccurred())

			testConsumerIP := testConsumerJob.Networks[0].StaticIPs[0]
			etcdClient = etcdclient.NewClient(fmt.Sprintf("http://%s:6769", testConsumerIP))
			spammer = helpers.NewSpammer(etcdClient, 1*time.Second)

			spammer.Spam()
		})

		By("deploy tls etcd, scale down non-tls etcd, deploy proxy, and switch clients to tls etcd", func() {
			var err error
			manifest, err = helpers.NewEtcdManifestWithTLSUpgrade(manifest.Name, client, config)
			Expect(err).NotTo(HaveOccurred())

			done := make(chan bool)
			go func() {
				err = helpers.ResolveVersionsAndDeploy(manifest, client)
				Expect(err).NotTo(HaveOccurred())
				done <- true
			}()

			Eventually(func() (string, error) {
				return getEtcdProxyState()
			}, "5m", "5s").Should(Equal("stopped"))

			Eventually(func() (string, error) {
				return getEtcdProxyState()
			}, "5m", "10s").Should(Equal("running"))

			spammer.ResetStore()

			<-done
			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
		})

		By("removing the proxy", func() {
			manifest = manifest.RemoveJob("etcd_z1")
			err := helpers.ResolveVersionsAndDeploy(manifest, client)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
		})

		By("stopping the spammer", func() {
			spammer.Stop()
		})

		By("reading from the cluster", func() {
			spammerErrors := spammer.Check()

			errorSet := spammerErrors.(helpers.ErrorSet)
			etcdErrorCount := 0
			testConsumerConnectionResetErrorCount := 0
			otherErrors := helpers.ErrorSet{}
			for err, _ := range errorSet {
				if strings.Contains(err, "last error: Put") {
					etcdErrorCount++
				} else if strings.Contains(err, "Unexpected HTTP status code") {
					etcdErrorCount++
				} else if strings.Contains(err, "EOF") {
					testConsumerConnectionResetErrorCount++
				} else {
					otherErrors.Add(errors.New(err))
				}
			}

			Expect(etcdErrorCount).To(BeNumerically("<", PUT_ERROR_COUNT_THRESHOLD))
			Expect(testConsumerConnectionResetErrorCount).To(BeNumerically("<", TEST_CONSUMER_CONNECTION_RESET_ERROR_COUNT))
			Expect(otherErrors).To(HaveLen(0))
		})
	})
})
