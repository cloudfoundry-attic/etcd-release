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
	etcdArgs := os.Args[3:]

	commandWrapper := command.NewWrapper(etcdPath, etcdArgs, os.Stdout, os.Stderr)

	logger := lager.NewLogger("etcdfab")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	app := application.New(application.NewArgs{
		Command:        commandWrapper,
		CommandPidPath: etcdPidPath,
	})
	err := app.Start()
	if err != nil {
		logger.Error("main", err)
		os.Exit(1)
	}
}
