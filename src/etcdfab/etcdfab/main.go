package main

import (
	"log"
	"os"
	"time"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/application"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/cluster"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/command"

	"code.cloudfoundry.org/lager"
)

func main() {
	etcdPidPath := os.Args[1]
	configFilePath := os.Args[2]
	linkConfigFilePath := os.Args[4]

	logger := lager.NewLogger("etcdfab")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	commandWrapper := command.NewWrapper()
	etcdClient := client.NewEtcdClient()
	clusterController := cluster.NewController(etcdClient, logger, time.Sleep)

	app := application.New(application.NewArgs{
		Command:            commandWrapper,
		CommandPidPath:     etcdPidPath,
		ConfigFilePath:     configFilePath,
		LinkConfigFilePath: linkConfigFilePath,
		EtcdClient:         etcdClient,
		ClusterController:  clusterController,
		OutWriter:          os.Stdout,
		ErrWriter:          os.Stderr,
		Logger:             logger,
	})

	err := app.Start()
	if err != nil {
		stderr := log.New(os.Stderr, "", 0)
		stderr.Printf("error during start: %s", err)
		os.Exit(1)
	}
}
