package fakes

import "io"

type CommandWrapper struct {
	StartCall struct {
		CallCount int
		Receives  struct {
			CommandPath string
			CommandArgs []string
			OutWriter   io.Writer
			ErrWriter   io.Writer
		}
		Returns struct {
			Pid   int
			Error error
		}
	}

	KillCall struct {
		CallCount int
		Receives  struct {
			Pid int
		}
		Returns struct {
			Error error
		}
	}
}

func (c *CommandWrapper) Start(commandPath string, commandArgs []string, outWriter, errWriter io.Writer) (int, error) {
	c.StartCall.CallCount++
	c.StartCall.Receives.CommandPath = commandPath
	c.StartCall.Receives.CommandArgs = commandArgs
	c.StartCall.Receives.OutWriter = outWriter
	c.StartCall.Receives.ErrWriter = errWriter

	return c.StartCall.Returns.Pid, c.StartCall.Returns.Error
}

func (c *CommandWrapper) Kill(pid int) error {
	c.KillCall.CallCount++
	c.KillCall.Receives.Pid = pid

	return c.KillCall.Returns.Error
}
