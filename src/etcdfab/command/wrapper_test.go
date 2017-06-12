package command_test

import (
	"os"
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
		It("kills the process", func() {
			cmd := exec.Command("yes")

			Expect(cmd.Start()).NotTo(HaveOccurred())

			pid := cmd.Process.Pid
			process, _ := os.FindProcess(pid)

			statusChan := make(chan string, 1)
			go func() {
				state, _ := process.Wait()
				statusChan <- state.String()
			}()

			commandWrapper := command.NewWrapper()
			Expect(commandWrapper.Kill(pid)).NotTo(HaveOccurred())

			var message string
			Eventually(statusChan).Should(Receive(&message))
			Expect(message).To(Equal("signal: killed"))
		})

		Context("when killing the process returns an error", func() {
			It("returns the error to the caller", func() {
				commandWrapper := command.NewWrapper()
				Expect(commandWrapper.Kill(12345)).To(MatchError(ContainSubstring("process already finished")))
			})
		})
	})
})
