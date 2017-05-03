package application

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"code.cloudfoundry.org/lager"
)

type Application struct {
	command        command
	commandPidPath string
	configFilePath string
	etcdPath       string
	etcdArgs       []string
	outWriter      io.Writer
	errWriter      io.Writer
}

type node struct {
	Name  string
	Index int
}

type config struct {
	Node node
}

type command interface {
	Start(string, []string, io.Writer, io.Writer) (int, error)
}

type logger interface {
	Error(string, error, ...lager.Data)
}

type NewArgs struct {
	Command        command
	CommandPidPath string
	ConfigFilePath string
	EtcdPath       string
	EtcdArgs       []string
	OutWriter      io.Writer
	ErrWriter      io.Writer
}

func New(args NewArgs) Application {
	return Application{
		command:        args.Command,
		commandPidPath: args.CommandPidPath,
		configFilePath: args.ConfigFilePath,
		etcdPath:       args.EtcdPath,
		etcdArgs:       args.EtcdArgs,
		outWriter:      args.OutWriter,
		errWriter:      args.ErrWriter,
	}
}

func (a Application) Start() error {
	configFileContents, err := ioutil.ReadFile(a.configFilePath)
	if err != nil {
		return err
	}

	var config config
	if err := json.Unmarshal(configFileContents, &config); err != nil {
		return err
	}

	a.etcdArgs = append(a.etcdArgs, "--name")
	a.etcdArgs = append(a.etcdArgs, fmt.Sprintf("%s-%d", strings.Replace(config.Node.Name, "_", "-", -1), config.Node.Index))

	pid, err := a.command.Start(a.etcdPath, a.etcdArgs, a.outWriter, a.errWriter)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(a.commandPidPath, []byte(fmt.Sprintf("%d", pid)), 0644)
	if err != nil {
		return err
	}

	return nil
}
