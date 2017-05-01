package application_test

import (
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/application"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Application", func() {
	Describe("Start", func() {
		var (
			etcdPidPath string
			fakeCommand *fakes.CommandWrapper

			app application.Application
		)

		BeforeEach(func() {
			fakeCommand = &fakes.CommandWrapper{}

			tmpDir, err := ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			etcdPidPath = fmt.Sprintf("%s/etcd-pid", tmpDir)

			app = application.New(application.NewArgs{
				CommandPidPath: etcdPidPath,
				Command:        fakeCommand,
			})
		})

		It("calls Start on the command", func() {
			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeCommand.StartCall.CallCount).To(Equal(1))
		})

		It("writes the pid of etcd to the file provided", func() {
			fakeCommand.Process.Pid = 12345

			err := app.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(etcdPidPath).To(BeARegularFile())

			etcdPid, err := ioutil.ReadFile(etcdPidPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(etcdPid)).To(Equal("12345"))
		})

		Context("failure cases", func() {
			Context("when commandWrapper.Start returns an error", func() {
				It("returns the error to the caller", func() {
					fakeCommand.StartCall.Returns.Error = errors.New("failed to start command")

					err := app.Start()
					Expect(err).To(MatchError("failed to start command"))
				})
			})

			Context("when it cannot write to the specified PID file", func() {
				It("returns the error to the caller", func() {
					fakeCommand.Process.Pid = 12345
					app = application.New(application.NewArgs{
						CommandPidPath: "/path/to/missing/file",
						Command:        fakeCommand,
					})

					err := app.Start()
					Expect(err).To(MatchError("open /path/to/missing/file: no such file or directory"))
				})
			})
		})
	})
})
