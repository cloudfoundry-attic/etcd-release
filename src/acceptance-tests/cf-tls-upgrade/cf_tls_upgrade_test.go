package cf_tls_upgrade_test

import (
	"acceptance-tests/cf-tls-upgrade/logspammer"
	"acceptance-tests/testing/helpers"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/cloudfoundry/noaa/consumer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
)

const (
	CF_PUSH_TIMEOUT = 2 * time.Minute
	DEFAULT_TIMEOUT = 30 * time.Second
)

var _ = PDescribe("CF TLS Upgrade Test", func() {
	var (
		appName string
		spammer *logspammer.Spammer
	)

	var getToken = func() string {
		session := cf.Cf("oauth-token")
		Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

		token := strings.TrimSpace(string(session.Out.Contents()))
		Expect(token).NotTo(Equal(""))
		return token
	}

	var getAppGuid = func(appName string) string {
		cfApp := cf.Cf("app", appName, "--guid")
		Eventually(cfApp, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

		appGuid := strings.TrimSpace(string(cfApp.Out.Contents()))
		Expect(appGuid).NotTo(Equal(""))
		return appGuid
	}

	var enableDiego = func(appName string) {
		guid := getAppGuid(appName)
		Eventually(cf.Cf("curl", "/v2/apps/"+guid, "-X", "PUT", "-d", `{"diego": true}`), DEFAULT_TIMEOUT).Should(gexec.Exit(0))
	}

	It("keeps writing logs for apps with little interruption", func() {
		By("targeting the cf deployment", func() {
			// cf.Target.....
		})

		By("push an app on Diego", func() {
			fmt.Println(appName)
			appName = generator.PrefixedRandomName("EATS-APP-")
			Eventually(cf.Cf(
				"push", appName,
				"-p", "assets/logspinner",
				"-f", "assets/logspinner/manifest.yml",
				"-i", "2",
				"-b", "go_buildpack",
				"--no-start"),
				CF_PUSH_TIMEOUT).Should(gexec.Exit(0))

			enableDiego(appName)

			Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(gexec.Exit(0))
		})

		By("spamming logs", func() {
			consumer := consumer.New("wss://doppler.bosh-lite.com:4443", &tls.Config{InsecureSkipVerify: true}, nil)
			msgChan, _ := consumer.Stream(getAppGuid(appName), getToken())
			spammer = logspammer.NewSpammer(fmt.Sprintf("http://%s.bosh-lite.com", appName), msgChan, 10*time.Millisecond)

			Eventually(func() bool {
				return spammer.CheckStream()
			}).Should(BeTrue())

			fmt.Println("starting spammer at ", time.Now())
			err := spammer.Start()
			Expect(err).NotTo(HaveOccurred())
		})

		By("deploy tls etcd, scale down non-tls etcd, deploy proxy, and switch clients to tls etcd", func() {
			fmt.Println("deploying cf at ", time.Now())
			rawManifest, err := client.DownloadManifest("cf-warden")
			Expect(err).NotTo(HaveOccurred())

			manifest, err := helpers.CreateCFTLSMigrationManifest(string(rawManifest))
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Deploy([]byte(manifest))
			Expect(err).NotTo(HaveOccurred())

			expectedVMs, err := helpers.GetVMsFromRawManifest(string(manifest))
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs("cf-warden")
			}, "1m", "10s").Should(ConsistOf(expectedVMs))
		})

		By("deploy diego to switch clients to tls etcd", func() {
			fmt.Println("deploying diego at ", time.Now())
			rawManifest, err := client.DownloadManifest("cf-warden-diego")
			Expect(err).NotTo(HaveOccurred())

			manifest, err := helpers.CreateDiegoTLSMigrationManifest(string(rawManifest))
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Deploy([]byte(manifest))
			Expect(err).NotTo(HaveOccurred())

			expectedVMs, err := helpers.GetVMsFromRawManifest(string(manifest))
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs("cf-warden-diego")
			}, "1m", "10s").Should(ConsistOf(expectedVMs))
		})

		By("removing proxy", func() {
			fmt.Println("deploying without proxy at ", time.Now())
			rawManifest, err := client.DownloadManifest("cf-warden")
			Expect(err).NotTo(HaveOccurred())

			manifest, err := helpers.RemoveEtcdProxy(string(rawManifest))
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Deploy([]byte(manifest))
			Expect(err).NotTo(HaveOccurred())

			expectedVMs, err := helpers.GetVMsFromRawManifest(string(manifest))
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs("cf-warden")
			}, "1m", "10s").Should(ConsistOf(expectedVMs))
		})

		By("stop spamming logs", func() {
			fmt.Println("stop spamming at ", time.Now())
			err := spammer.Stop()
			Expect(err).NotTo(HaveOccurred())
		})

		By("verify minimal log loss", func() {
			err := spammer.Check()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
