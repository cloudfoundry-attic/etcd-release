package cf_tls_upgrade_test

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/etcd-release/src/acceptance-tests/cf-tls-upgrade/logspammer"
	"github.com/cloudfoundry-incubator/etcd-release/src/acceptance-tests/cf-tls-upgrade/syslogchecker"
	"github.com/cloudfoundry-incubator/etcd-release/src/acceptance-tests/testing/helpers"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/cloudfoundry/sonde-go/events"

	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/ops"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	CF_PUSH_TIMEOUT                       = 2 * time.Minute
	DEFAULT_TIMEOUT                       = 30 * time.Second
	GUID_NOT_FOUND_ERROR_THRESHOLD        = 1
	GATEWAY_TIMEOUT_ERROR_COUNT_THRESHOLD = 2
	BAD_GATEWAY_ERROR_COUNT_THRESHOLD     = 2
	MISSING_LOG_THRESHOLD                 = 200 // Frequency of spammer is 100ms (allow 20s of missing logs)
)

type gen struct{}

func (gen) Generate() string {
	return strconv.Itoa(rand.Int())
}

type runner struct{}

func (runner) Run(args ...string) ([]byte, error) {
	return exec.Command("cf", args...).CombinedOutput()
}

var _ = Describe("CF TLS Upgrade Test", func() {
	It("successfully upgrades etcd cluster to use TLS", func() {
		var (
			nonTLSManifest   string
			etcdTLSManifest  string
			proxyTLSManifest string
			manifestName     string

			err     error
			appName string

			spammer *logspammer.Spammer
			checker syslogchecker.Checker
		)

		varsStoreBytes, err := ioutil.ReadFile(config.BOSH.DeploymentVarsPath)
		Expect(err).NotTo(HaveOccurred())
		varsStore := string(varsStoreBytes)

		var findValue = func(path string) string {
			value, err := ops.FindOp(varsStore, path)
			Expect(err).NotTo(HaveOccurred())
			return value.(string)
		}

		etcdClientCA := findValue("/etcd_client/ca")
		etcdClientCertificate := findValue("/etcd_client/certificate")
		etcdClientPrivateKey := findValue("/etcd_client/private_key")

		etcdServerCA := findValue("/etcd_server/ca")
		etcdServerCertificate := findValue("/etcd_server/certificate")
		etcdServerPrivateKey := findValue("/etcd_server/private_key")

		etcdPeerCA := findValue("/etcd_peer/ca")
		etcdPeerCertificate := findValue("/etcd_peer/certificate")
		etcdPeerPrivateKey := findValue("/etcd_peer/private_key")

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

		By("checking if the expected non-tls VMs are running", func() {
			byteManifest, err := boshClient.DownloadManifest("cf")
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile("original-non-tls-manifest.yml", byteManifest, 0644)
			Expect(err).NotTo(HaveOccurred())

			nonTLSManifest = string(byteManifest)
			manifestName, err = ops.ManifestName(nonTLSManifest)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]bosh.VM, error) {
				return helpers.DeploymentVMs(boshClient, manifestName)
			}, "1m", "10s").Should(ConsistOf(helpers.GetNonErrandVMsFromManifest(nonTLSManifest)))
		})

		By("logging into cf and preparing the environment", func() {
			Eventually(
				cf.Cf("login", "-a", fmt.Sprintf("api.%s", config.CF.Domain),
					"-u", config.CF.Username, "-p", config.CF.Password,
					"--skip-ssl-validation"),
				DEFAULT_TIMEOUT).Should(gexec.Exit(0))

			Eventually(cf.Cf("create-org", "EATS_org"), DEFAULT_TIMEOUT).Should(gexec.Exit(0))
			Eventually(cf.Cf("target", "-o", "EATS_org"), DEFAULT_TIMEOUT).Should(gexec.Exit(0))

			Eventually(cf.Cf("create-space", "EATS_space"), DEFAULT_TIMEOUT).Should(gexec.Exit(0))
			Eventually(cf.Cf("target", "-s", "EATS_space"), DEFAULT_TIMEOUT).Should(gexec.Exit(0))

			Eventually(cf.Cf("enable-feature-flag", "diego_docker"), DEFAULT_TIMEOUT).Should(gexec.Exit(0))
		})

		By("pushing an application to diego", func() {
			appName = generator.PrefixedRandomName("EATS-APP-", "")
			Eventually(cf.Cf(
				"push", appName,
				"-f", "assets/logspinner/manifest.yml",
				"--no-start"),
				CF_PUSH_TIMEOUT).Should(gexec.Exit(0))

			enableDiego(appName)

			Eventually(cf.Cf("start", appName), CF_PUSH_TIMEOUT).Should(gexec.Exit(0))
		})

		By("starting the syslog-drain process", func() {
			syslogAppName := generator.PrefixedRandomName("syslog-source-app-", "")
			Eventually(cf.Cf(
				"push", syslogAppName,
				"-f", "assets/logspinner/manifest.yml",
				"--no-start"),
				CF_PUSH_TIMEOUT).Should(gexec.Exit(0))

			enableDiego(syslogAppName)

			Eventually(cf.Cf("start", syslogAppName), CF_PUSH_TIMEOUT).Should(gexec.Exit(0))
			checker = syslogchecker.New("syslog-drainer", gen{}, 1*time.Millisecond, runner{})
			checker.Start(syslogAppName, fmt.Sprintf("http://%s.%s", syslogAppName, config.CF.Domain))
		})

		By("spamming logs", func() {
			consumer := consumer.New(fmt.Sprintf("wss://doppler.%s:443", config.CF.Domain), &tls.Config{InsecureSkipVerify: true}, nil)

			spammer = logspammer.NewSpammer(os.Stdout, fmt.Sprintf("http://%s.%s", appName, config.CF.Domain),
				func() (<-chan *events.Envelope, <-chan error) {
					return consumer.Stream(getAppGuid(appName), getToken())
				},
				100*time.Millisecond,
			)

			Eventually(func() bool {
				return spammer.CheckStream()
			}, "10s", "1s").Should(BeTrue())

			err = spammer.Start()
			Expect(err).NotTo(HaveOccurred())
		})

		By("deploying a TLS etcd cluster", func() {
			etcd, err := ops.FindOp(nonTLSManifest, "/instance_groups/name=etcd")
			Expect(err).NotTo(HaveOccurred())

			etcdTLSManifest, err = ops.ApplyOps(nonTLSManifest, []ops.Op{
				// # --- Add an etcd-tls group, keep the etcd group ---
				{"replace", "/instance_groups/name=etcd/name", "etcd-non-tls"},
				{"replace", "/instance_groups/-", etcd},
				{"replace", "/instance_groups/name=etcd/name", "etcd-tls"},
				{"replace", "/instance_groups/name=etcd-non-tls/name", "etcd"},
			})

			etcdTLSManifest, err = ops.ApplyOps(etcdTLSManifest, []ops.Op{
				// # --- Add consul agent job ---
				{
					Type: "replace",
					Path: "/instance_groups/name=etcd-tls/jobs/-",
					Value: map[string]interface{}{
						"name":    "consul_agent",
						"release": "consul",
						"consumes": map[string]interface{}{
							"consul": map[string]string{
								"from": "consul_server",
							},
						},
						"properties": map[string]interface{}{
							"consul": map[string]interface{}{
								"agent": map[string]interface{}{
									"services": map[string]interface{}{
										"etcd": map[string]string{
											"name": "cf-etcd",
										},
									},
								},
							},
						},
					},
				},

				// # --- Add cluster properties ---
				{
					Type: "replace",
					Path: "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/cluster?/-",
					Value: map[string]interface{}{
						"name":      "etcd",
						"instances": 3,
					},
				},

				// # --- Remove static ips for the etcd-tls machines ---
				{"remove", "/instance_groups/name=etcd-tls/networks/name=default/static_ips", ""},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/machines", []string{"cf-etcd.service.cf.internal"}},

				// # --- Enable ssl requirements and add certs/keys on etcd-tls ---
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/peer_require_ssl", true},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/require_ssl", true},

				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/ca_cert?", etcdClientCA},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/client_cert?", etcdClientCertificate},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/client_key?", etcdClientPrivateKey},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/server_cert?", etcdServerCertificate},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/server_key?", etcdServerPrivateKey},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/peer_ca_cert?", etcdPeerCA},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/peer_cert?", etcdPeerCertificate},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd/properties/etcd/peer_key?", etcdPeerPrivateKey},

				// # --- Remove static ip of etcd instance for the etcd metrics server ---
				{"remove", "/instance_groups/name=etcd-tls/jobs/name=etcd_metrics_server/properties/etcd_metrics_server/etcd/machine", ""},

				// # --- Enable tls communication for etcd_metrics_server-etcd ---
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd_metrics_server/properties/etcd_metrics_server/etcd/require_ssl", true},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd_metrics_server/properties/etcd_metrics_server/etcd/ca_cert?", etcdServerCA},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd_metrics_server/properties/etcd_metrics_server/etcd/client_cert?", etcdClientCertificate},
				{"replace", "/instance_groups/name=etcd-tls/jobs/name=etcd_metrics_server/properties/etcd_metrics_server/etcd/client_key?", etcdClientPrivateKey},
			})
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile("add-tls-etcd-deploy-manifest.yml", []byte(etcdTLSManifest), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = boshClient.Deploy([]byte(etcdTLSManifest))
			Expect(err).NotTo(HaveOccurred())
		})

		By("checking if the expected etcd-tls VMs are running", func() {
			Eventually(func() ([]bosh.VM, error) {
				return helpers.DeploymentVMs(boshClient, manifestName)
			}, "1m", "10s").Should(ConsistOf(helpers.GetNonErrandVMsFromManifest(etcdTLSManifest)))
		})

		By("scaling down the non-TLS etcd cluster to 1 node and converting it to a proxy", func() {
			proxyTLSManifest, err = ops.ApplyOps(etcdTLSManifest, []ops.Op{
				// --- Rename job to etcd_proxy ---
				{"replace", "/instance_groups/name=etcd/jobs/name=etcd/name", "etcd_proxy"},
				// --- Scale etcd_proxy down to 1 instance ---
				{"replace", "/instance_groups/name=etcd/instances", 1},
				// --- Add static ips for the etcd machines ---
				{"replace", "/instance_groups/name=etcd/networks/name=default/static_ips?", []string{"10.0.31.231"}},
				// --- Reduce etcd_proxy to 1 az ---
				{"replace", "/instance_groups/name=etcd/azs", []string{"z1"}},
				// --- Add consul agent without advertising consul service ---
				{
					Type: "replace",
					Path: "/instance_groups/name=etcd/jobs/-",
					Value: map[string]interface{}{
						"name":    "consul_agent",
						"release": "consul",
						"consumes": map[string]interface{}{
							"consul": map[string]string{
								"from": "consul_server",
							},
						},
					},
				},
				// Set the etcd_proxy properties ---
				{"replace", "/instance_groups/name=etcd/jobs/name=etcd_proxy/properties/etcd_proxy?/etcd/dns_suffix", "cf-etcd.service.cf.internal"},
				{"replace", "/instance_groups/name=etcd/jobs/name=etcd_proxy/properties/etcd_proxy/etcd/ca_cert?", etcdClientCA},
				{"replace", "/instance_groups/name=etcd/jobs/name=etcd_proxy/properties/etcd_proxy/etcd/client_cert?", etcdClientCertificate},
				{"replace", "/instance_groups/name=etcd/jobs/name=etcd_proxy/properties/etcd_proxy/etcd/client_key?", etcdClientPrivateKey},
				{"remove", "/instance_groups/name=etcd/jobs/name=etcd_proxy/properties/etcd", ""},
				// --- Remove the etcd metrics server from the proxy ---
				{"remove", "/instance_groups/name=etcd/jobs/name=etcd_metrics_server", ""},

				// # --- Add etcd properties on all metron agent jobs ---
				{
					Type: "replace",
					Path: "/instance_groups/name=consul/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=nats/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=etcd-tls/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=mysql/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=uaa/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=blobstore/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=api/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=cc-worker/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=router/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=diego-bbs/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=diego-brain/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=diego-cell/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=route-emitter/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=cc-clock/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=cc-bridge/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=doppler/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{
					Type: "replace",
					Path: "/instance_groups/name=log-api/jobs/name=metron_agent/properties/metron_agent/etcd?",
					Value: map[string]string{
						"client_cert": etcdClientCertificate,
						"client_key":  etcdClientPrivateKey,
					},
				},
				{"replace", "/instance_groups/name=diego-bbs/jobs/name=bbs/properties/diego/bbs/etcd/require_ssl?", false},
				{"replace", "/instance_groups/name=diego-bbs/jobs/name=bbs/properties/diego/bbs/etcd/machines", nil},

				// # --- Enable tls communication for doppler-etcd ---
				{"replace", "/instance_groups/name=doppler/jobs/name=doppler/properties/doppler/etcd/client_cert?", etcdClientCertificate},
				{"replace", "/instance_groups/name=doppler/jobs/name=doppler/properties/doppler/etcd/client_key?", etcdClientPrivateKey},
				{"replace", "/instance_groups/name=doppler/jobs/name=doppler/properties/loggregator/etcd/require_ssl?", true},
				{"replace", "/instance_groups/name=doppler/jobs/name=doppler/properties/loggregator/etcd/ca_cert?", etcdServerCA},
				{"replace", "/instance_groups/name=doppler/jobs/name=doppler/properties/loggregator/etcd/machines?", []string{"cf-etcd.service.cf.internal"}},

				{"replace", "/instance_groups/name=doppler/jobs/name=syslog_drain_binder/properties/loggregator/etcd/require_ssl?", true},
				{"replace", "/instance_groups/name=doppler/jobs/name=syslog_drain_binder/properties/loggregator/etcd/ca_cert?", etcdServerCA},
				{"replace", "/instance_groups/name=doppler/jobs/name=syslog_drain_binder/properties/loggregator/etcd/machines", []string{"cf-etcd.service.cf.internal"}},
				{"replace", "/instance_groups/name=doppler/jobs/name=syslog_drain_binder/properties/syslog_drain_binder/etcd/client_cert?", etcdClientCertificate},
				{"replace", "/instance_groups/name=doppler/jobs/name=syslog_drain_binder/properties/syslog_drain_binder/etcd/client_key?", etcdClientPrivateKey},

				// # --- Enable tls communication for log api-etcd ---
				{"replace", "/instance_groups/name=log-api/jobs/name=loggregator_trafficcontroller/properties/traffic_controller/etcd/client_cert?", etcdClientCertificate},
				{"replace", "/instance_groups/name=log-api/jobs/name=loggregator_trafficcontroller/properties/traffic_controller/etcd/client_key?", etcdClientPrivateKey},
				{"replace", "/instance_groups/name=log-api/jobs/name=loggregator_trafficcontroller/properties/loggregator/etcd/require_ssl?", true},
				{"replace", "/instance_groups/name=log-api/jobs/name=loggregator_trafficcontroller/properties/loggregator/etcd/ca_cert?", etcdServerCA},
				{"replace", "/instance_groups/name=log-api/jobs/name=loggregator_trafficcontroller/properties/loggregator/etcd/machines?", []string{"cf-etcd.service.cf.internal"}},
			})
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile("proxy-etcd-deploy-manifest.yml", []byte(proxyTLSManifest), 0644)
			Expect(err).NotTo(HaveOccurred())

			_, err = boshClient.Deploy([]byte(proxyTLSManifest))
			Expect(err).NotTo(HaveOccurred())
		})

		By("checking if the expected proxy-tls VMs are running", func() {
			Eventually(func() ([]bosh.VM, error) {
				return helpers.DeploymentVMs(boshClient, manifestName)
			}, "1m", "10s").Should(ConsistOf(helpers.GetNonErrandVMsFromManifest(proxyTLSManifest)))
		})

		By("stopping spammer and checking for errors", func() {
			err = spammer.Stop()
			Expect(err).NotTo(HaveOccurred())

			spammerErrs, missingLogErrors := spammer.Check()

			var errorSet helpers.ErrorSet

			switch spammerErrs.(type) {
			case helpers.ErrorSet:
				errorSet = spammerErrs.(helpers.ErrorSet)
			case nil:
			default:
				Fail(spammerErrs.Error())
			}

			badGatewayErrCount := 0
			gatewayTimeoutErrCount := 0
			otherErrors := helpers.ErrorSet{}

			for err, occurrences := range errorSet {
				switch {
				// This typically happens when an active connection to a cell is interrupted during a cell evacuation
				case strings.Contains(err, "504 GATEWAY_TIMEOUT"):
					gatewayTimeoutErrCount += occurrences
				// This typically happens when an active connection to a cell is interrupted during a cell evacuation
				case strings.Contains(err, "502 Bad Gateway"):
					badGatewayErrCount += occurrences
				default:
					otherErrors.Add(errors.New(err))
				}
			}

			var missingLogErrorsCount int
			if missingLogErrors != nil {
				missingLogErrorsCount = len(missingLogErrors.(helpers.ErrorSet))
				if missingLogErrorsCount > MISSING_LOG_THRESHOLD {
					fmt.Println(missingLogErrors)
				}
			}

			Expect(otherErrors).To(HaveLen(0))
			Expect(missingLogErrorsCount).To(BeNumerically("<=", MISSING_LOG_THRESHOLD))
			Expect(gatewayTimeoutErrCount).To(BeNumerically("<=", GATEWAY_TIMEOUT_ERROR_COUNT_THRESHOLD))
			Expect(badGatewayErrCount).To(BeNumerically("<=", BAD_GATEWAY_ERROR_COUNT_THRESHOLD))
		})

		By("running a couple iterations of the syslog-drain checker", func() {
			count := checker.GetIterationCount()
			Eventually(checker.GetIterationCount, "10m", "10s").Should(BeNumerically(">", count+2))
		})

		By("stopping syslogchecker and checking for errors", func() {
			err = checker.Stop()
			Expect(err).NotTo(HaveOccurred())

			if ok, iterationCount, errPercent, errs := checker.Check(); ok {
				fmt.Println("total errors were within threshold")
				fmt.Println("total iterations:", iterationCount)
				fmt.Println("error percentage:", errPercent)
				fmt.Println(errs)
			} else {
				Fail(errs.Error())
			}
		})
	})
})
