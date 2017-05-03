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

type configNode struct {
	Name  string
	Index int
}

type configEtcd struct {
	HeartbeatInterval int    `json:"heartbeat_interval_in_milliseconds"`
	ElectionTimeout   int    `json:"election_timeout_in_milliseconds"`
	PeerRequireSSL    bool   `json:"peer_require_ssl"`
	PeerIP            string `json:"peer_ip"`
	RequireSSL        bool   `json:"require_ssl"`
	ClientIP          string `json:"client_ip"`
}

type config struct {
	Node configNode
	Etcd configEtcd
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
	a.etcdArgs = append(a.etcdArgs, "--data-dir")
	a.etcdArgs = append(a.etcdArgs, "/var/vcap/store/etcd")
	a.etcdArgs = append(a.etcdArgs, "--heartbeat-interval")
	a.etcdArgs = append(a.etcdArgs, fmt.Sprintf("%d", config.Etcd.HeartbeatInterval))
	a.etcdArgs = append(a.etcdArgs, "--election-timeout")
	a.etcdArgs = append(a.etcdArgs, fmt.Sprintf("%d", config.Etcd.ElectionTimeout))

	peerProtocol := "http"
	if config.Etcd.PeerRequireSSL {
		peerProtocol = "https"
	}
	a.etcdArgs = append(a.etcdArgs, "--listen-peer-urls")
	a.etcdArgs = append(a.etcdArgs, fmt.Sprintf("%s://%s:7001", peerProtocol, config.Etcd.PeerIP))

	clientProtocol := "http"
	if config.Etcd.RequireSSL {
		clientProtocol = "https"
	}
	a.etcdArgs = append(a.etcdArgs, "--listen-client-urls")
	a.etcdArgs = append(a.etcdArgs, fmt.Sprintf("%s://%s:4001", clientProtocol, config.Etcd.ClientIP))

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
