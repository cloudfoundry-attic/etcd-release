package application

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"

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
	logger         logger
}

type command interface {
	Start(string, []string, io.Writer, io.Writer) (int, error)
}

type logger interface {
	Info(string, ...lager.Data)
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
	Logger         logger
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
		logger:         args.Logger,
	}
}

func (a Application) Start() error {
	cfg, err := config.ConfigFromJSON(a.configFilePath)
	if err != nil {
		a.logger.Error("application.read-config-file.failed", err)
		return err
	}

	nodeName := fmt.Sprintf("%s-%d", strings.Replace(cfg.Node.Name, "_", "-", -1), cfg.Node.Index)
	a.logger.Info("application.build-etcd-flags", lager.Data{"node-name": nodeName})

	peerProtocol := "http"
	if cfg.Etcd.PeerRequireSSL {
		peerProtocol = "https"
	}

	clientProtocol := "http"
	if cfg.Etcd.RequireSSL {
		clientProtocol = "https"
	}

	peerUrl := fmt.Sprintf("http://%s:7001", cfg.Node.ExternalIP)
	if cfg.Etcd.PeerRequireSSL || cfg.Etcd.RequireSSL {
		peerUrl = fmt.Sprintf("https://%s.%s:7001", nodeName, cfg.Etcd.AdvertiseURLsDNSSuffix)
	}

	clientUrl := fmt.Sprintf("http://%s:4001", cfg.Node.ExternalIP)
	if cfg.Etcd.PeerRequireSSL || cfg.Etcd.RequireSSL {
		clientUrl = fmt.Sprintf("https://%s.%s:4001", nodeName, cfg.Etcd.AdvertiseURLsDNSSuffix)
	}

	a.etcdArgs = append(a.etcdArgs, "--name")
	a.etcdArgs = append(a.etcdArgs, nodeName)

	a.etcdArgs = append(a.etcdArgs, "--data-dir")
	a.etcdArgs = append(a.etcdArgs, "/var/vcap/store/etcd")

	a.etcdArgs = append(a.etcdArgs, "--heartbeat-interval")
	a.etcdArgs = append(a.etcdArgs, fmt.Sprintf("%d", cfg.Etcd.HeartbeatInterval))

	a.etcdArgs = append(a.etcdArgs, "--election-timeout")
	a.etcdArgs = append(a.etcdArgs, fmt.Sprintf("%d", cfg.Etcd.ElectionTimeout))

	a.etcdArgs = append(a.etcdArgs, "--listen-peer-urls")
	a.etcdArgs = append(a.etcdArgs, fmt.Sprintf("%s://%s:7001", peerProtocol, cfg.Etcd.PeerIP))

	a.etcdArgs = append(a.etcdArgs, "--listen-client-urls")
	a.etcdArgs = append(a.etcdArgs, fmt.Sprintf("%s://%s:4001", clientProtocol, cfg.Etcd.ClientIP))

	a.etcdArgs = append(a.etcdArgs, "--initial-advertise-peer-urls")
	a.etcdArgs = append(a.etcdArgs, peerUrl)

	a.etcdArgs = append(a.etcdArgs, "--advertise-client-urls")
	a.etcdArgs = append(a.etcdArgs, clientUrl)

	pid, err := a.command.Start(a.etcdPath, a.etcdArgs, a.outWriter, a.errWriter)
	if err != nil {
		a.logger.Error("application.start.failed", err)
		return err
	}

	err = ioutil.WriteFile(a.commandPidPath, []byte(fmt.Sprintf("%d", pid)), 0644)
	if err != nil {
		a.logger.Error("application.write-pid-file.failed", err)
		return err
	}

	return nil
}
