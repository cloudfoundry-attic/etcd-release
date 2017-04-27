package main

import (
	"os"
	"os/exec"

	"code.cloudfoundry.org/lager"
)

func main() {
	etcdPath := os.Args[1]
	etcdArgs := os.Args[2:]

	logger := lager.NewLogger("etcdfab")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	etcdCommand := exec.Command(etcdPath, etcdArgs...)
	etcdCommand.Stdout = os.Stdout
	etcdCommand.Stderr = os.Stderr

	err := etcdCommand.Run()
	if err != nil {
		logger.Error("main", err)
		os.Exit(1)
	}
}
