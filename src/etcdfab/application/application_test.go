package application_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/application"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Application", func() {
	Describe("Start", func() {
		var (
			etcdPidPath    string
			configFileName string
			fakeCommand    *fakes.CommandWrapper

			outWriter bytes.Buffer
			errWriter bytes.Buffer

			app application.Application
		)

		BeforeEach(func() {
			fakeCommand = &fakes.CommandWrapper{}

			fakeCommand.StartCall.Returns.Pid = 12345

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
				EtcdPath:       "path-to-etcd",
				EtcdArgs:       []string{"arg-1", "arg-2"},
				OutWriter:      &outWriter,
				ErrWriter:      &errWriter,
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

		It("writes the pid of etcd to the file provided", func() {
			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(etcdPidPath).To(BeARegularFile())

			etcdPid, err := ioutil.ReadFile(etcdPidPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(etcdPid)).To(Equal("12345"))
		})

		Context("failure cases", func() {
			Context("when commandWrapper.Start returns an error", func() {
				BeforeEach(func() {
					fakeCommand.StartCall.Returns.Error = errors.New("failed to start command")
				})

				It("returns the error to the caller", func() {
					err := app.Start()
					Expect(err).To(MatchError("failed to start command"))
				})
			})

			Context("when it cannot write to the specified PID file", func() {
				BeforeEach(func() {
					app = application.New(application.NewArgs{
						Command:        fakeCommand,
						CommandPidPath: "/path/to/missing/file",
						ConfigFilePath: configFileName,
					})
				})

				It("returns the error to the caller", func() {
					err := app.Start()
					Expect(err).To(MatchError("open /path/to/missing/file: no such file or directory"))
				})
			})

			Context("when it cannot read the config file", func() {
				BeforeEach(func() {
					app = application.New(application.NewArgs{
						Command:        fakeCommand,
						CommandPidPath: etcdPidPath,
						ConfigFilePath: "/path/to/missing/file",
					})
				})

				It("returns the error to the caller", func() {
					err := app.Start()
					Expect(err).To(MatchError("open /path/to/missing/file: no such file or directory"))
				})
			})

			Context("when it cannot unmarshal the config file", func() {
				BeforeEach(func() {
					err := ioutil.WriteFile(configFileName, []byte("%%%"), os.ModePerm)
					Expect(err).NotTo(HaveOccurred())

					app = application.New(application.NewArgs{
						Command:        fakeCommand,
						CommandPidPath: etcdPidPath,
						ConfigFilePath: configFileName,
					})
				})

				It("returns the error to the caller", func() {
					err := app.Start()
					Expect(err).To(MatchError("invalid character '%' looking for beginning of value"))
				})
			})
		})
	})
})
