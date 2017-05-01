package command_test

import (
	"bytes"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/command"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Wrapper", func() {
	Describe("NewWrapper", func() {
		It("returns a command object with the provided path", func() {
			commandWrapper := command.NewWrapper("/path/to/command", []string{}, nil, nil)

			Expect(commandWrapper.ExecCmd.Path).To(Equal("/path/to/command"))
		})

		It("returns a command object with the provided args", func() {
			commandWrapper := command.NewWrapper("/path/to/command", []string{"arg1", "arg2"}, nil, nil)

			Expect(commandWrapper.ExecCmd.Args).To(Equal([]string{"/path/to/command", "arg1", "arg2"}))
		})

		It("returns a command object with Stdout connected to the provided output writer", func() {
			var outWriter bytes.Buffer

			commandWrapper := command.NewWrapper("", []string{}, &outWriter, nil)

			Expect(commandWrapper.ExecCmd.Stdout).To(Equal(&outWriter))
		})

		It("returns a command object with Stderr connected to the provided error writer", func() {
			var errWriter bytes.Buffer

			commandWrapper := command.NewWrapper("", []string{}, nil, &errWriter)

			Expect(commandWrapper.ExecCmd.Stderr).To(Equal(&errWriter))
		})
	})

	Describe("Start", func() {
		It("runs a command", func() {
			var (
				outWriter bytes.Buffer
				errWriter bytes.Buffer
			)

			commandWrapper := command.NewWrapper("echo", []string{"hello"}, &outWriter, &errWriter)
			err := commandWrapper.Start()
			Expect(err).NotTo(HaveOccurred())

			err = commandWrapper.ExecCmd.Wait()
			Expect(err).NotTo(HaveOccurred())

			Expect(outWriter.String()).To(Equal("hello\n"))
			Expect(errWriter.String()).To(Equal(""))
		})

		Context("when exec.Cmd.Start returns an error", func() {
			It("returns the error to the caller", func() {
				commandWrapper := command.NewWrapper("bogus", []string{}, nil, nil)
				err := commandWrapper.Start()
				Expect(err).To(MatchError(ContainSubstring("executable file not found in $PATH")))
			})
		})
	})

	Describe("GetProcessID", func() {
		It("returns the process ID of the command being run", func() {
			commandWrapper := command.NewWrapper("echo", []string{"hello"}, nil, nil)
			err := commandWrapper.Start()
			Expect(err).NotTo(HaveOccurred())

			Expect(commandWrapper.GetProcessID()).To(SatisfyAll(
				BeNumerically(">", 0),
				BeNumerically("<", 4194304),
			))

			err = commandWrapper.ExecCmd.Wait()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
