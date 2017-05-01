package fakes

import "io"

type CommandWrapper struct {
	StartCall struct {
		CallCount int
		Returns   struct {
			Error error
		}
	}
	Process struct {
		Pid int
	}
	Stdout io.Writer
	Stderr io.Writer
}

func (c *CommandWrapper) Start() error {
	c.StartCall.CallCount++

	return c.StartCall.Returns.Error
}

func (c *CommandWrapper) GetProcessID() int {
	return c.Process.Pid
}
