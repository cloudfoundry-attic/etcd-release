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

var _ = Describe("CF TLS Upgrade Test", func() {
	It("successfully upgrades etcd cluster to use TLS", func() {
		var (
			migrationManifest []byte
			err               error
			appName           string
			spammer           *logspammer.Spammer
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

		By("logging into cf and preparing the environment", func() {
			cfConfig := config.CF
			Eventually(
				cf.Cf("login", "-a", fmt.Sprintf("api.%s", cfConfig.Domain),
					"-u", cfConfig.Username, "-p", cfConfig.Password,
					"--skip-ssl-validation"),
				DEFAULT_TIMEOUT).Should(gexec.Exit(0))

			Eventually(cf.Cf("create-org", "EATS_org"), DEFAULT_TIMEOUT).Should(gexec.Exit(0))
			Eventually(cf.Cf("target", "-o", "EATS_org"), DEFAULT_TIMEOUT).Should(gexec.Exit(0))

			Eventually(cf.Cf("create-space", "EATS_space"), DEFAULT_TIMEOUT).Should(gexec.Exit(0))
			Eventually(cf.Cf("target", "-s", "EATS_space"), DEFAULT_TIMEOUT).Should(gexec.Exit(0))

			Eventually(cf.Cf("enable-feature-flag", "diego_docker"), DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		})

		By("pushing an application to diego", func() {
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
			consumer := consumer.New(fmt.Sprintf("wss://doppler.%s:4443", config.CF.Domain), &tls.Config{InsecureSkipVerify: true}, nil)
			msgChan, _ := consumer.Stream(getAppGuid(appName), getToken())
			spammer = logspammer.NewSpammer(fmt.Sprintf("http://%s.%s", appName, config.CF.Domain), msgChan, 10*time.Millisecond)
			Eventually(func() bool {
				return spammer.CheckStream()
			}).Should(BeTrue())

			err = spammer.Start()
			Expect(err).NotTo(HaveOccurred())
		})

		By("scaling down the non-TLS etcd cluster to 1 node and converting it to a proxy", func() {
			originalManifest, err := client.DownloadManifest(config.BOSH.DeploymentName)
			Expect(err).NotTo(HaveOccurred())

			migrationManifest, err = helpers.CreateCFTLSMigrationManifest(originalManifest)
			Expect(err).NotTo(HaveOccurred())

			_, err = client.Deploy(migrationManifest)
			Expect(err).NotTo(HaveOccurred())
		})

		By("checking if expected VMs are running", func() {
			expectedVMs, err := helpers.GetNonErrandVMsFromRawManifest(migrationManifest)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return client.DeploymentVMs(config.BOSH.DeploymentName)
			}, "1m", "10s").Should(ConsistOf(expectedVMs))
		})

		By("stopping spammer and checking for errors", func() {
			err = spammer.Stop()
			Expect(err).NotTo(HaveOccurred())

			err = spammer.Check()
			Expect(err).NotTo(HaveOccurred())
		})

	})
})
