package application

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/client"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/cluster"
	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"

	"code.cloudfoundry.org/lager"
)

type Application struct {
	command            command
	configFilePath     string
	linkConfigFilePath string
	etcdClient         etcdClient
	clusterController  clusterController
	syncController     syncController
	outWriter          io.Writer
	errWriter          io.Writer
	logger             logger
}

type command interface {
	Start(string, []string, io.Writer, io.Writer) (int, error)
	Kill(int) error
}

type syncController interface {
	VerifySynced() error
}

type clusterController interface {
	GetInitialClusterState(config.Config) (cluster.InitialClusterState, error)
}

type etcdClient interface {
	Configure(client.Config) error
	MemberRemove(string) error
	MemberList() ([]client.Member, error)
}

type logger interface {
	Info(string, ...lager.Data)
	Error(string, error, ...lager.Data)
}

type NewArgs struct {
	Command            command
	ConfigFilePath     string
	LinkConfigFilePath string
	EtcdClient         etcdClient
	ClusterController  clusterController
	SyncController     syncController
	OutWriter          io.Writer
	ErrWriter          io.Writer
	Logger             logger
}

func New(args NewArgs) Application {
	return Application{
		command:            args.Command,
		configFilePath:     args.ConfigFilePath,
		linkConfigFilePath: args.LinkConfigFilePath,
		etcdClient:         args.EtcdClient,
		clusterController:  args.ClusterController,
		syncController:     args.SyncController,
		outWriter:          args.OutWriter,
		errWriter:          args.ErrWriter,
		logger:             args.Logger,
	}
}

func (a Application) Start() error {
	cfg, err := config.ConfigFromJSONs(a.configFilePath, a.linkConfigFilePath)
	if err != nil {
		a.logger.Error("application.read-config-file.failed", err)
		return err
	}

	err = a.etcdClient.Configure(cfg)
	if err != nil {
		a.logger.Error("application.etcd-client.configure.failed", err)
		return err
	}

	initialClusterState, err := a.clusterController.GetInitialClusterState(cfg)
	if err != nil {
		a.logger.Error("application.cluster-controller.get-initial-cluster-state.failed", err)
		return err
	}

	etcdArgs := a.buildEtcdArgs(cfg)

	etcdArgs = append(etcdArgs, "--initial-cluster")
	etcdArgs = append(etcdArgs, initialClusterState.Members)
	etcdArgs = append(etcdArgs, "--initial-cluster-state")
	etcdArgs = append(etcdArgs, initialClusterState.State)

	a.logger.Info("application.start", lager.Data{
		"etcd-path": cfg.Etcd.EtcdPath,
		"etcd-args": etcdArgs,
	})
	pid, err := a.command.Start(cfg.Etcd.EtcdPath, etcdArgs, a.outWriter, a.errWriter)
	if err != nil {
		a.logger.Error("application.start.failed", err)
		return err
	}

	a.logger.Info("application.synchronized-controller.verify-synced")
	syncErr := a.syncController.VerifySynced()
	if syncErr != nil {
		a.logger.Error("application.synchronized-controller.verify-synced.failed", syncErr)

		if initialClusterState.State == "existing" {
			a.logger.Info("application.remove-self-from-cluster")
			a.removeSelfFromCluster(cfg)
		}
		a.removeDataDir(cfg)

		a.logger.Info("application.kill")
		killErr := a.kill(cfg.PidFile())
		if killErr != nil {
			return killErr
		}
		return syncErr
	}

	a.logger.Info("application.write-pid-file", lager.Data{
		"pid":  pid,
		"path": cfg.PidFile(),
	})
	err = ioutil.WriteFile(cfg.PidFile(), []byte(fmt.Sprintf("%d", pid)), 0644)
	if err != nil {
		a.logger.Error("application.write-pid-file.failed", err)
		return err
	}

	a.logger.Info("application.start.success")

	return nil
}

func (a Application) Stop() error {
	a.logger.Info("application.stop")

	cfg, err := config.ConfigFromJSONs(a.configFilePath, a.linkConfigFilePath)
	if err != nil {
		a.logger.Error("application.read-config-file.failed", err)
		return err
	}

	err = a.etcdClient.Configure(cfg)
	if err != nil {
		a.logger.Error("application.etcd-client.configure.failed", err)
		return err
	}

	teardown := a.priorClusterHadOtherNodes(cfg.NodeName())
	if teardown {
		a.logger.Info("application.remove-self-from-cluster")
		a.removeSelfFromCluster(cfg)
	}

	a.removeDataDir(cfg)

	a.logger.Info("application.kill")
	err = a.kill(cfg.PidFile())
	if err != nil {
		return err
	}

	a.logger.Info("application.stop.success")
	return nil
}

func (a Application) priorClusterHadOtherNodes(nodeName string) bool {
	a.logger.Info("application.etcd-client.member-list")
	memberList, err := a.etcdClient.MemberList()
	if err != nil {
		a.logger.Error("application.etcd-client.member-list.failed", err)
		return false
	}

	a.logger.Info("application.etcd-client.member-list", lager.Data{"member-list": memberList})
	if len(memberList) == 1 && memberList[0].Name == nodeName {
		return false
	}

	if len(memberList) == 1 && memberList[0].Name != nodeName {
		return true
	}

	if len(memberList) > 1 {
		return true
	}

	return false
}

func (a Application) removeSelfFromCluster(cfg config.Config) {
	memberList, err := a.etcdClient.MemberList()
	if err != nil {
		a.logger.Error("application.etcd-client.member-list.failed", err)
	}
	var memberID string
	for _, member := range memberList {
		if member.Name == cfg.NodeName() {
			memberID = member.ID
		}
	}

	a.logger.Info("application.etcd-client.member-remove", lager.Data{"member-id": memberID})
	err = a.etcdClient.MemberRemove(memberID)
	if err != nil {
		a.logger.Error("application.etcd-client.member-remove.failed", err)
	}
}

func (a Application) removeDataDir(cfg config.Config) {
	a.logger.Info("application.remove-data-dir", lager.Data{"data-dir": cfg.Etcd.DataDir})
	d, err := os.Open(cfg.Etcd.DataDir)
	if err != nil {
		a.logger.Error("application.remove-data-dir", err)
	}
	defer d.Close()
	files, err := d.Readdirnames(-1)
	if err != nil {
		a.logger.Error("application.remove-data-dir", err)
	}
	for _, file := range files {
		err = os.RemoveAll(filepath.Join(cfg.Etcd.DataDir, file))
	}
	if err != nil {
		a.logger.Error("application.remove-data-dir", err)
	}
}

func (a Application) kill(pidPath string) error {
	a.logger.Info("application.read-pid-file", lager.Data{"pid-file": pidPath})
	pidFileContents, err := ioutil.ReadFile(pidPath)
	if err != nil {
		a.logger.Error("application.read-pid-file.failed", err)
		return err
	}

	a.logger.Info("application.convert-pid-file-to-pid")
	pid, err := strconv.Atoi(string(pidFileContents))
	if err != nil {
		a.logger.Error("application.convert-pid-file-to-pid.failed", err)
		return err
	}

	a.logger.Info("application.kill-pid", lager.Data{"pid": pid})
	err = a.command.Kill(pid)
	if err != nil {
		a.logger.Error("application.kill-pid.failed", err)
		return err
	}

	a.logger.Info("application.remove-pid-file")
	err = os.Remove(pidPath)
	if err != nil {
		//not tested
		a.logger.Error("application.remove-pid-file.failed", err)
		return err
	}

	return nil
}

func (a Application) buildEtcdArgs(cfg config.Config) []string {
	a.logger.Info("application.build-etcd-flags", lager.Data{"node-name": cfg.NodeName()})

	var etcdArgs []string
	etcdArgs = append(etcdArgs, "--name")
	etcdArgs = append(etcdArgs, cfg.NodeName())

	if cfg.Etcd.EnableDebugLogging {
		etcdArgs = append(etcdArgs, "--debug")
	}

	etcdArgs = append(etcdArgs, "--data-dir")
	etcdArgs = append(etcdArgs, cfg.Etcd.DataDir)

	etcdArgs = append(etcdArgs, "--heartbeat-interval")
	etcdArgs = append(etcdArgs, fmt.Sprintf("%d", cfg.Etcd.HeartbeatInterval))

	etcdArgs = append(etcdArgs, "--election-timeout")
	etcdArgs = append(etcdArgs, fmt.Sprintf("%d", cfg.Etcd.ElectionTimeout))

	etcdArgs = append(etcdArgs, "--listen-peer-urls")
	etcdArgs = append(etcdArgs, cfg.ListenPeerURL())

	etcdArgs = append(etcdArgs, "--listen-client-urls")
	etcdArgs = append(etcdArgs, cfg.ListenClientURL())

	etcdArgs = append(etcdArgs, "--initial-advertise-peer-urls")
	etcdArgs = append(etcdArgs, cfg.AdvertisePeerURL())

	etcdArgs = append(etcdArgs, "--advertise-client-urls")
	etcdArgs = append(etcdArgs, cfg.AdvertiseClientURL())

	if cfg.Etcd.RequireSSL {
		etcdArgs = append(etcdArgs, "--client-cert-auth")
		etcdArgs = append(etcdArgs, "--trusted-ca-file")
		etcdArgs = append(etcdArgs, filepath.Join(cfg.CertDir(), "server-ca.crt"))
		etcdArgs = append(etcdArgs, "--cert-file")
		etcdArgs = append(etcdArgs, filepath.Join(cfg.CertDir(), "server.crt"))
		etcdArgs = append(etcdArgs, "--key-file")
		etcdArgs = append(etcdArgs, filepath.Join(cfg.CertDir(), "server.key"))
	}

	if cfg.Etcd.PeerRequireSSL {
		etcdArgs = append(etcdArgs, "--peer-client-cert-auth")
		etcdArgs = append(etcdArgs, "--peer-trusted-ca-file")
		etcdArgs = append(etcdArgs, filepath.Join(cfg.CertDir(), "peer-ca.crt"))
		etcdArgs = append(etcdArgs, "--peer-cert-file")
		etcdArgs = append(etcdArgs, filepath.Join(cfg.CertDir(), "peer.crt"))
		etcdArgs = append(etcdArgs, "--peer-key-file")
		etcdArgs = append(etcdArgs, filepath.Join(cfg.CertDir(), "peer.key"))
	}

	return etcdArgs
}
