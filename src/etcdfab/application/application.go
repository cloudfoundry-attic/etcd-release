package application

import (
	"fmt"
	"io/ioutil"

	"code.cloudfoundry.org/lager"
)

type Application struct {
	command        command
	commandPidPath string
}

type command interface {
	Start() error
	GetProcessID() int
}

type logger interface {
	Error(string, error, ...lager.Data)
}

type NewArgs struct {
	Command        command
	CommandPidPath string
}

func New(args NewArgs) Application {
	return Application{
		command:        args.Command,
		commandPidPath: args.CommandPidPath,
	}
}

func (a Application) Start() error {
	err := a.command.Start()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(a.commandPidPath, []byte(fmt.Sprintf("%d", a.command.GetProcessID())), 0644)
	if err != nil {
		return err
	}

	return nil
}
