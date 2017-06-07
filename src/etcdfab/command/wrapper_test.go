package command_test

import (
	"os/exec"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/command"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wrapper", func() {
	Describe("Start", func() {
		It("runs a command and returns the process id", func() {
			outWriter := newConcurrentSafeBuffer()
			errWriter := newConcurrentSafeBuffer()

			commandWrapper := command.NewWrapper()
			pid, err := commandWrapper.Start("echo", []string{"hello"}, outWriter, errWriter)
			Expect(err).NotTo(HaveOccurred())

			Expect(pid).To(SatisfyAll(
				BeNumerically(">", 0),
				BeNumerically("<", 4194304),
			))

			Eventually(outWriter.String).Should(Equal("hello\n"))
			Expect(errWriter.String()).To(Equal(""))
		})

		Context("when exec.Cmd.Start returns an error", func() {
			It("returns the error to the caller", func() {
				commandWrapper := command.NewWrapper()
				_, err := commandWrapper.Start("bogus", []string{}, nil, nil)
				Expect(err).To(MatchError(ContainSubstring("executable file not found in $PATH")))
			})
		})
	})

	Describe("Kill", func() {
		var pid int

		BeforeEach(func() {
			cmd := exec.Command("echo", "./fake-process.sh")

			err := cmd.Start()
			Expect(err).NotTo(HaveOccurred())

			pid = cmd.Process.Pid
		})

		It("kills the process", func() {
			commandWrapper := command.NewWrapper()
			err := commandWrapper.Kill(pid)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when killing the process returns an error", func() {
			It("returns the error to the caller", func() {
				commandWrapper := command.NewWrapper()
				err := commandWrapper.Kill(-1)
				Expect(err).To(MatchError(ContainSubstring("process already released")))
			})
		})
	})
})
