package main

import (
	"os"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/application"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/command"

	"code.cloudfoundry.org/lager"
)

// var generateCommand func(name string, arg ...string) *Cmd
// var generateCommand func(name string, arg ...string) *exec.Cmd
// var generateCommand = exec.Command

func main() {
	etcdPath := os.Args[1]
	etcdPidPath := os.Args[2]
	configFilePath := os.Args[3]
	etcdArgs := os.Args[4:]

	commandWrapper := command.NewWrapper()

	logger := lager.NewLogger("etcdfab")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	app := application.New(application.NewArgs{
		Command:        commandWrapper,
		CommandPidPath: etcdPidPath,
		ConfigFilePath: configFilePath,
		EtcdPath:       etcdPath,
		EtcdArgs:       etcdArgs,
		OutWriter:      os.Stdout,
		ErrWriter:      os.Stderr,
	})

	err := app.Start()
	if err != nil {
		logger.Error("main", err)
		os.Exit(1)
	}
}
