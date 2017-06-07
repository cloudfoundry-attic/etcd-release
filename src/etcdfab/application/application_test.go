package application_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/application"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/cluster"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/gomegamatchers"
)

func createConfig(tmpDir, name string, configuration map[string]interface{}) string {
	file, err := ioutil.TempFile(tmpDir, name)
	Expect(err).NotTo(HaveOccurred())

	err = file.Close()
	Expect(err).NotTo(HaveOccurred())

	fileName := file.Name()

	configData, err := json.Marshal(configuration)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(fileName, configData, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return fileName
}

var _ = Describe("Application", func() {
	Describe("Start", func() {
		var (
			tmpDir             string
			runDir             string
			configFileName     string
			linkConfigFileName string

			etcdfabConfig config.Config

			fakeCommand                *fakes.CommandWrapper
			fakeClusterController      *fakes.ClusterController
			fakeSynchronizedController *fakes.SynchronizedController
			fakeEtcdClient             *fakes.EtcdClient
			certDir                    string
			fakeLogger                 *fakes.Logger

			outWriter bytes.Buffer
			errWriter bytes.Buffer

			app application.Application
		)

		BeforeEach(func() {
			fakeCommand = &fakes.CommandWrapper{}
			fakeCommand.StartCall.Returns.Pid = 12345

			fakeEtcdClient = &fakes.EtcdClient{}
			fakeClusterController = &fakes.ClusterController{}
			fakeClusterController.GetInitialClusterStateCall.Returns.InitialClusterState = cluster.InitialClusterState{
				Members: "etcd-0=http://some-ip-1:7001",
				State:   "new",
			}

			fakeSynchronizedController = &fakes.SynchronizedController{}

			fakeLogger = &fakes.Logger{}

			var err error
			tmpDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			runDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			configuration := map[string]interface{}{
				"node": map[string]interface{}{
					"name":        "some_name",
					"index":       3,
					"external_ip": "some-external-ip",
				},
				"etcd": map[string]interface{}{
					"etcd_path": "path-to-etcd",
					"cert_dir":  "some/cert/dir",
					"run_dir":   runDir,
					"heartbeat_interval_in_milliseconds": 10,
					"election_timeout_in_milliseconds":   20,
					"peer_require_ssl":                   false,
					"peer_ip":                            "some-peer-ip",
					"require_ssl":                        false,
					"client_ip":                          "some-client-ip",
					"advertise_urls_dns_suffix":          "some-dns-suffix",
				},
			}
			configFileName = createConfig(tmpDir, "config-file", configuration)

			linkConfiguration := map[string]interface{}{
				"machines": []string{"some-ip-1", "some-ip-2"},
			}
			linkConfigFileName = createConfig(tmpDir, "config-link-file", linkConfiguration)

			etcdfabConfig = config.Config{
				Node: config.Node{
					Name:       "some_name",
					Index:      3,
					ExternalIP: "some-external-ip",
				},
				Etcd: config.Etcd{
					EtcdPath:               "path-to-etcd",
					CertDir:                "some/cert/dir",
					RunDir:                 runDir,
					HeartbeatInterval:      10,
					ElectionTimeout:        20,
					PeerRequireSSL:         false,
					PeerIP:                 "some-peer-ip",
					RequireSSL:             false,
					ClientIP:               "some-client-ip",
					AdvertiseURLsDNSSuffix: "some-dns-suffix",
					Machines:               []string{"some-ip-1", "some-ip-2"},
				},
			}

			app = application.New(application.NewArgs{
				Command:                fakeCommand,
				ConfigFilePath:         configFileName,
				LinkConfigFilePath:     linkConfigFileName,
				EtcdClient:             fakeEtcdClient,
				CertDir:                certDir,
				ClusterController:      fakeClusterController,
				SynchronizedController: fakeSynchronizedController,
				OutWriter:              &outWriter,
				ErrWriter:              &errWriter,
				Logger:                 fakeLogger,
			})
		})

		It("configures the etcd client", func() {
			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeEtcdClient.ConfigureCall.CallCount).To(Equal(1))
			Expect(fakeEtcdClient.ConfigureCall.Receives.Config).To(Equal(etcdfabConfig))
		})

		It("calls Start on the command", func() {
			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCommand.StartCall.CallCount).To(Equal(1))
			Expect(fakeCommand.StartCall.Receives.CommandPath).To(Equal("path-to-etcd"))
			Expect(fakeCommand.StartCall.Receives.CommandArgs).To(Equal([]string{
				"--name", "some-name-3",
				"--data-dir", "/var/vcap/store/etcd",
				"--heartbeat-interval", "10",
				"--election-timeout", "20",
				"--listen-peer-urls", "http://some-peer-ip:7001",
				"--listen-client-urls", "http://some-client-ip:4001",
				"--initial-advertise-peer-urls", "http://some-external-ip:7001",
				"--advertise-client-urls", "http://some-external-ip:4001",
				"--initial-cluster", "etcd-0=http://some-ip-1:7001",
				"--initial-cluster-state", "new",
			}))
			Expect(fakeCommand.StartCall.Receives.OutWriter).To(Equal(&outWriter))
			Expect(fakeCommand.StartCall.Receives.ErrWriter).To(Equal(&errWriter))
		})

		It("calls GetInitialCluster and GetInitialClusterState on the cluster controller", func() {
			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeClusterController.GetInitialClusterStateCall.CallCount).To(Equal(1))
			Expect(fakeClusterController.GetInitialClusterStateCall.Receives.Config).To(Equal(etcdfabConfig))

			Expect(fakeCommand.StartCall.Receives.CommandArgs).To(Equal([]string{
				"--name", "some-name-3",
				"--data-dir", "/var/vcap/store/etcd",
				"--heartbeat-interval", "10",
				"--election-timeout", "20",
				"--listen-peer-urls", "http://some-peer-ip:7001",
				"--listen-client-urls", "http://some-client-ip:4001",
				"--initial-advertise-peer-urls", "http://some-external-ip:7001",
				"--advertise-client-urls", "http://some-external-ip:4001",
				"--initial-cluster", "etcd-0=http://some-ip-1:7001",
				"--initial-cluster-state", "new",
			}))
		})

		It("verifies the cluster is synced", func() {
			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeSynchronizedController.VerifySyncedCall.CallCount).To(Equal(1))
		})

		It("writes the pid of etcd to the run dir", func() {
			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			etcdPidPath := filepath.Join(runDir, "etcd.pid")
			Expect(etcdPidPath).To(BeARegularFile())

			etcdPid, err := ioutil.ReadFile(etcdPidPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(etcdPid)).To(Equal("12345"))
		})

		Context("when configured to be a tls etcd cluster", func() {
			BeforeEach(func() {
				configuration := map[string]interface{}{
					"node": map[string]interface{}{
						"name":        "some_name",
						"index":       3,
						"external_ip": "some-external-ip",
					},
					"etcd": map[string]interface{}{
						"etcd_path": "path-to-etcd",
						"cert_dir":  "some/cert/dir",
						"run_dir":   runDir,
						"heartbeat_interval_in_milliseconds": 10,
						"election_timeout_in_milliseconds":   20,
						"peer_require_ssl":                   true,
						"peer_ip":                            "some-peer-ip",
						"require_ssl":                        true,
						"client_ip":                          "some-client-ip",
						"advertise_urls_dns_suffix":          "some-dns-suffix",
						"ca_cert":                            "some-ca-cert",
						"server_cert":                        "some-server-cert",
						"server_key":                         "some-server-key",
						"peer_ca_cert":                       "some-peer-ca-cert",
						"peer_cert":                          "some-peer-cert",
						"peer_key":                           "some-peer-key",
					},
				}
				configData, err := json.Marshal(configuration)
				Expect(err).NotTo(HaveOccurred())

				err = ioutil.WriteFile(configFileName, configData, os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				app = application.New(application.NewArgs{
					Command:                fakeCommand,
					ConfigFilePath:         configFileName,
					LinkConfigFilePath:     linkConfigFileName,
					EtcdClient:             fakeEtcdClient,
					ClusterController:      fakeClusterController,
					SynchronizedController: fakeSynchronizedController,
					OutWriter:              &outWriter,
					ErrWriter:              &errWriter,
					Logger:                 fakeLogger,
				})
			})

			It("calls Start on the command with etcd security flags", func() {
				err := app.Start()
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCommand.StartCall.CallCount).To(Equal(1))
				Expect(fakeCommand.StartCall.Receives.CommandPath).To(Equal("path-to-etcd"))
				Expect(fakeCommand.StartCall.Receives.CommandArgs).To(Equal([]string{
					"--name", "some-name-3",
					"--data-dir", "/var/vcap/store/etcd",
					"--heartbeat-interval", "10",
					"--election-timeout", "20",
					"--listen-peer-urls", "https://some-peer-ip:7001",
					"--listen-client-urls", "https://some-client-ip:4001",
					"--initial-advertise-peer-urls", "https://some-name-3.some-dns-suffix:7001",
					"--advertise-client-urls", "https://some-name-3.some-dns-suffix:4001",
					"--client-cert-auth",
					"--trusted-ca-file", "some/cert/dir/server-ca.crt",
					"--cert-file", "some/cert/dir/server.crt",
					"--key-file", "some/cert/dir/server.key",
					"--peer-client-cert-auth",
					"--peer-trusted-ca-file", "some/cert/dir/peer-ca.crt",
					"--peer-cert-file", "some/cert/dir/peer.crt",
					"--peer-key-file", "some/cert/dir/peer.key",
					"--initial-cluster", "etcd-0=http://some-ip-1:7001",
					"--initial-cluster-state", "new",
				}))
				Expect(fakeCommand.StartCall.Receives.OutWriter).To(Equal(&outWriter))
				Expect(fakeCommand.StartCall.Receives.ErrWriter).To(Equal(&errWriter))
			})
		})

		It("writes informational log messages", func() {
			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeLogger.Messages()).To(ConsistOf([]fakes.LoggerMessage{
				{
					Action: "application.build-etcd-flags",
					Data: []lager.Data{{
						"node-name": "some-name-3",
					}},
				},
				{
					Action: "application.start",
					Data: []lager.Data{{
						"etcd-path": "path-to-etcd",
						"etcd-args": []string{
							"--name", "some-name-3",
							"--data-dir", "/var/vcap/store/etcd",
							"--heartbeat-interval", "10",
							"--election-timeout", "20",
							"--listen-peer-urls", "http://some-peer-ip:7001",
							"--listen-client-urls", "http://some-client-ip:4001",
							"--initial-advertise-peer-urls", "http://some-external-ip:7001",
							"--advertise-client-urls", "http://some-external-ip:4001",
							"--initial-cluster", "etcd-0=http://some-ip-1:7001",
							"--initial-cluster-state", "new",
						},
					}},
				},
			}))
		})

		Context("failure cases", func() {
			Context("when it cannot read the config file", func() {
				BeforeEach(func() {
					app = application.New(application.NewArgs{
						ConfigFilePath:     "/path/to/missing/file",
						LinkConfigFilePath: linkConfigFileName,
						Logger:             fakeLogger,
					})
				})

				It("returns the error to the caller and logs a helpful message", func() {
					err := app.Start()
					Expect(err).To(MatchError("error reading config file: open /path/to/missing/file: no such file or directory"))

					Expect(fakeLogger.Messages()).To(ConsistOf([]fakes.LoggerMessage{
						{
							Action: "application.read-config-file.failed",
							Error:  err,
						},
					}))
				})
			})

			Context("when it cannot read the link config file", func() {
				BeforeEach(func() {
					app = application.New(application.NewArgs{
						ConfigFilePath:     configFileName,
						LinkConfigFilePath: "/path/to/missing/file",
						Logger:             fakeLogger,
					})
				})

				It("returns the error to the caller and logs a helpful message", func() {
					err := app.Start()
					Expect(err).To(MatchError("error reading link config file: open /path/to/missing/file: no such file or directory"))

					Expect(fakeLogger.Messages()).To(ConsistOf([]fakes.LoggerMessage{
						{
							Action: "application.read-config-file.failed",
							Error:  err,
						},
					}))
				})
			})

			Context("when etcdClient.Configure returns an error", func() {
				BeforeEach(func() {
					fakeEtcdClient.ConfigureCall.Returns.Error = errors.New("failed to configure etcd client")
				})

				It("returns the error to the caller and logs a helpful message", func() {
					err := app.Start()
					Expect(err).To(MatchError("failed to configure etcd client"))

					Expect(fakeLogger.Messages()).To(gomegamatchers.ContainSequence([]fakes.LoggerMessage{
						{
							Action: "application.etcd-client.configure.failed",
							Error:  err,
						},
					}))
				})
			})

			Context("when clusterController.GetInitialClusterState returns an error", func() {
				BeforeEach(func() {
					fakeClusterController.GetInitialClusterStateCall.Returns.Error = errors.New("failed to get initial cluster state")
				})

				It("returns the error to the caller and logs a helpful message", func() {
					err := app.Start()
					Expect(err).To(MatchError("failed to get initial cluster state"))

					Expect(fakeLogger.Messages()).To(gomegamatchers.ContainSequence([]fakes.LoggerMessage{
						{
							Action: "application.cluster-controller.get-initial-cluster-state.failed",
							Error:  err,
						},
					}))
				})
			})

			Context("when commandWrapper.Start returns an error", func() {
				BeforeEach(func() {
					fakeCommand.StartCall.Returns.Error = errors.New("failed to start command")
				})

				It("returns the error to the caller and logs a helpful message", func() {
					err := app.Start()
					Expect(err).To(MatchError("failed to start command"))

					Expect(fakeLogger.Messages()).To(gomegamatchers.ContainSequence([]fakes.LoggerMessage{
						{
							Action: "application.start.failed",
							Error:  err,
						},
					}))
				})
			})

			Context("when synchronizeController.VerifySynced returns an error", func() {
				BeforeEach(func() {
					fakeSynchronizedController.VerifySyncedCall.Returns.Error = errors.New("failed to verify synced")
				})

				It("returns an error to the caller and logs a helpful message", func() {
					err := app.Start()
					Expect(err).To(MatchError("failed to verify synced"))

					Expect(fakeLogger.Messages()).To(gomegamatchers.ContainSequence([]fakes.LoggerMessage{
						{
							Action: "application.synchronized-controller.verify-synced.failed",
							Error:  err,
						},
					}))
				})
			})

			Context("when it cannot write to the specified PID file", func() {
				BeforeEach(func() {
					configuration := map[string]interface{}{
						"etcd": map[string]interface{}{
							"run_dir": "/path/to/missing",
						},
					}
					configFileName = createConfig(tmpDir, "config-file", configuration)
					app = application.New(application.NewArgs{
						Command:                fakeCommand,
						ConfigFilePath:         configFileName,
						LinkConfigFilePath:     linkConfigFileName,
						EtcdClient:             fakeEtcdClient,
						ClusterController:      fakeClusterController,
						SynchronizedController: fakeSynchronizedController,
						Logger:                 fakeLogger,
					})
				})

				It("returns the error to the caller and logs a helpful message", func() {
					err := app.Start()
					Expect(err).To(MatchError("open /path/to/missing/etcd.pid: no such file or directory"))

					Expect(fakeLogger.Messages()).To(gomegamatchers.ContainSequence([]fakes.LoggerMessage{
						{
							Action: "application.write-pid-file.failed",
							Error:  err,
						},
					}))
				})
			})
		})
	})
})
