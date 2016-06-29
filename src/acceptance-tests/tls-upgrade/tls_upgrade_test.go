package tls_upgrade_test

import (
	etcdclient "acceptance-tests/testing/etcd"
	"acceptance-tests/testing/helpers"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/etcd"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	PUT_ERROR_COUNT_THRESHOLD                  = 10
	TEST_CONSUMER_CONNECTION_RESET_ERROR_COUNT = 1
)

var _ = Describe("TLS Upgrade", func() {
	var (
		manifest etcd.Manifest
		spammers []*helpers.Spammer
	)

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
			manifest, err = helpers.NewEtcdWithInstanceCount(3, client, config, false)
			Expect(err).NotTo(HaveOccurred())

			manifest, err = helpers.SetTestConsumerInstanceCount(5, manifest)
			Expect(err).NotTo(HaveOccurred())

			err = helpers.ResolveVersionsAndDeploy(manifest, client)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(manifest.Name)
			}, "1m", "10s").Should(ConsistOf(helpers.GetVMsFromManifest(manifest)))
		})

		By("spamming the cluster", func() {
			testConsumerJobIndex, err := helpers.FindJobIndexByName(manifest, "testconsumer_z1")
			Expect(err).NotTo(HaveOccurred())

			for i, ip := range manifest.Jobs[testConsumerJobIndex].Networks[0].StaticIPs {
				etcdClient := etcdclient.NewClient(fmt.Sprintf("http://%s:6769", ip))
				spammer := helpers.NewSpammer(etcdClient, 1*time.Second, fmt.Sprintf("tls-upgrade-%d", i))
				spammers = append(spammers, spammer)

				spammer.Spam()
			}
		})

		By("deploy tls etcd, scale down non-tls etcd, deploy proxy, and switch clients to tls etcd", func() {
			var err error
			manifest, err = helpers.NewEtcdManifestWithTLSUpgrade(manifest.Name, client, config)
			Expect(err).NotTo(HaveOccurred())

			done := make(chan bool)
			go func() {
				err := helpers.ResolveVersionsAndDeploy(manifest, client)
				Expect(err).NotTo(HaveOccurred())
				done <- true
			}()

			Eventually(func() (string, error) {
				return getEtcdProxyState()
			}, "8m", "5s").Should(Equal("stopped"))

			Eventually(func() (string, error) {
				return getEtcdProxyState()
			}, "8m", "5s").Should(Equal("running"))

			for _, spammer := range spammers {
				spammer.ResetStore()
			}

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
			for _, spammer := range spammers {
				spammer.Stop()
			}
		})

		By("reading from the cluster", func() {
			for _, spammer := range spammers {
				spammerErrors := spammer.Check()

				errorSet := spammerErrors.(helpers.ErrorSet)

				etcdErrorCount := 0
				testConsumerConnectionResetErrorCount := 0
				otherErrors := helpers.ErrorSet{}

				for err, occurrences := range errorSet {
					switch {
					case strings.Contains(err, "last error: Put"):
						etcdErrorCount += occurrences
					case strings.Contains(err, "EOF"):
						testConsumerConnectionResetErrorCount += occurrences
					default:
						otherErrors.Add(errors.New(err))
					}
				}

				Expect(etcdErrorCount).To(BeNumerically("<=", PUT_ERROR_COUNT_THRESHOLD))
				Expect(testConsumerConnectionResetErrorCount).To(BeNumerically("<=", TEST_CONSUMER_CONNECTION_RESET_ERROR_COUNT))
				Expect(otherErrors).To(HaveLen(0))
			}
		})
	})
})
