package command

import (
	"io"
	"os/exec"
)

type Wrapper struct {
	ExecCmd *exec.Cmd
}

func NewWrapper(commandPath string, commandArgs []string, outWriter, errWriter io.Writer) Wrapper {
	cmd := exec.Command(commandPath, commandArgs...)

	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	return Wrapper{
		ExecCmd: cmd,
	}
}

func (w Wrapper) Start() error {
	return w.ExecCmd.Start()
}

func (w Wrapper) GetProcessID() int {
	return w.ExecCmd.Process.Pid
}
