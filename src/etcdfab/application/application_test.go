package application_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/lager"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/application"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/gomegamatchers"
)

var _ = Describe("Application", func() {
	Describe("Start", func() {
		var (
			etcdPidPath    string
			configFileName string
			fakeCommand    *fakes.CommandWrapper
			fakeLogger     *fakes.Logger

			outWriter bytes.Buffer
			errWriter bytes.Buffer

			app application.Application
		)

		BeforeEach(func() {
			fakeCommand = &fakes.CommandWrapper{}
			fakeCommand.StartCall.Returns.Pid = 12345

			fakeLogger = &fakes.Logger{}

			tmpDir, err := ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			etcdPidPath = fmt.Sprintf("%s/etcd-pid", tmpDir)

			configFile, err := ioutil.TempFile(tmpDir, "config-file")
			Expect(err).NotTo(HaveOccurred())

			err = configFile.Close()
			Expect(err).NotTo(HaveOccurred())

			configFileName = configFile.Name()

			configuration := map[string]interface{}{
				"node": map[string]interface{}{
					"name":        "some_name",
					"index":       3,
					"external_ip": "some-external-ip",
				},
				"etcd": map[string]interface{}{
					"etcd_path":                          "path-to-etcd",
					"heartbeat_interval_in_milliseconds": 10,
					"election_timeout_in_milliseconds":   20,
					"peer_require_ssl":                   false,
					"peer_ip":                            "some-peer-ip",
					"require_ssl":                        false,
					"client_ip":                          "some-client-ip",
					"advertise_urls_dns_suffix":          "some-dns-suffix",
				},
			}
			configData, err := json.Marshal(configuration)
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(configFileName, configData, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			app = application.New(application.NewArgs{
				Command:        fakeCommand,
				CommandPidPath: etcdPidPath,
				ConfigFilePath: configFileName,
				EtcdArgs:       []string{"arg-1", "arg-2"},
				OutWriter:      &outWriter,
				ErrWriter:      &errWriter,
				Logger:         fakeLogger,
			})
		})

		It("calls Start on the command", func() {
			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCommand.StartCall.CallCount).To(Equal(1))
			Expect(fakeCommand.StartCall.Receives.CommandPath).To(Equal("path-to-etcd"))
			Expect(fakeCommand.StartCall.Receives.CommandArgs).To(Equal([]string{
				"arg-1",
				"arg-2",
				"--name", "some-name-3",
				"--data-dir", "/var/vcap/store/etcd",
				"--heartbeat-interval", "10",
				"--election-timeout", "20",
				"--listen-peer-urls", "http://some-peer-ip:7001",
				"--listen-client-urls", "http://some-client-ip:4001",
				"--initial-advertise-peer-urls", "http://some-external-ip:7001",
				"--advertise-client-urls", "http://some-external-ip:4001",
			}))
			Expect(fakeCommand.StartCall.Receives.OutWriter).To(Equal(&outWriter))
			Expect(fakeCommand.StartCall.Receives.ErrWriter).To(Equal(&errWriter))
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
						"etcd_path":                          "path-to-etcd",
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
					Command:        fakeCommand,
					CommandPidPath: etcdPidPath,
					ConfigFilePath: configFileName,
					EtcdArgs:       []string{"arg-1", "arg-2"},
					OutWriter:      &outWriter,
					ErrWriter:      &errWriter,
					Logger:         fakeLogger,
				})

			})

			It("calls Start on the command with etcd security flags", func() {
				err := app.Start()
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeCommand.StartCall.CallCount).To(Equal(1))
				Expect(fakeCommand.StartCall.Receives.CommandPath).To(Equal("path-to-etcd"))
				Expect(fakeCommand.StartCall.Receives.CommandArgs).To(Equal([]string{
					"arg-1",
					"arg-2",
					"--name", "some-name-3",
					"--data-dir", "/var/vcap/store/etcd",
					"--heartbeat-interval", "10",
					"--election-timeout", "20",
					"--listen-peer-urls", "https://some-peer-ip:7001",
					"--listen-client-urls", "https://some-client-ip:4001",
					"--initial-advertise-peer-urls", "https://some-name-3.some-dns-suffix:7001",
					"--advertise-client-urls", "https://some-name-3.some-dns-suffix:4001",
					"--client-cert-auth",
					"--trusted-ca-file", "/var/vcap/jobs/etcd/config/certs/server-ca.crt",
					"--cert-file", "/var/vcap/jobs/etcd/config/certs/server.crt",
					"--key-file", "/var/vcap/jobs/etcd/config/certs/server.key",
					"--peer-client-cert-auth",
					"--peer-trusted-ca-file", "/var/vcap/jobs/etcd/config/certs/peer-ca.crt",
					"--peer-cert-file", "/var/vcap/jobs/etcd/config/certs/peer.crt",
					"--peer-key-file", "/var/vcap/jobs/etcd/config/certs/peer.key",
				}))
				Expect(fakeCommand.StartCall.Receives.OutWriter).To(Equal(&outWriter))
				Expect(fakeCommand.StartCall.Receives.ErrWriter).To(Equal(&errWriter))
			})
		})

		It("writes the pid of etcd to the file provided", func() {
			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(etcdPidPath).To(BeARegularFile())

			etcdPid, err := ioutil.ReadFile(etcdPidPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(etcdPid)).To(Equal("12345"))
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
							"arg-1",
							"arg-2",
							"--name", "some-name-3",
							"--data-dir", "/var/vcap/store/etcd",
							"--heartbeat-interval", "10",
							"--election-timeout", "20",
							"--listen-peer-urls", "http://some-peer-ip:7001",
							"--listen-client-urls", "http://some-client-ip:4001",
							"--initial-advertise-peer-urls", "http://some-external-ip:7001",
							"--advertise-client-urls", "http://some-external-ip:4001",
						},
					}},
				},
			}))
		})

		Context("failure cases", func() {
			Context("when it cannot read the config file", func() {
				BeforeEach(func() {
					app = application.New(application.NewArgs{
						Command:        fakeCommand,
						CommandPidPath: etcdPidPath,
						ConfigFilePath: "/path/to/missing/file",
						Logger:         fakeLogger,
					})
				})

				It("returns the error to the caller and logs a helpful message", func() {
					err := app.Start()
					Expect(err).To(MatchError("open /path/to/missing/file: no such file or directory"))

					Expect(fakeLogger.Messages()).To(ConsistOf([]fakes.LoggerMessage{
						{
							Action: "application.read-config-file.failed",
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

			Context("when it cannot write to the specified PID file", func() {
				BeforeEach(func() {
					app = application.New(application.NewArgs{
						Command:        fakeCommand,
						CommandPidPath: "/path/to/missing/file",
						ConfigFilePath: configFileName,
						Logger:         fakeLogger,
					})
				})

				It("returns the error to the caller and logs a helpful message", func() {
					err := app.Start()
					Expect(err).To(MatchError("open /path/to/missing/file: no such file or directory"))

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
