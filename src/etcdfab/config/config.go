package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

const (
	clientPort      = 4001
	peerPort        = 7001
	etcdPidFilename = "etcd.pid"
)

type Node struct {
	Name       string
	Index      int
	ExternalIP string `json:"external_ip"`
}

type Etcd struct {
	EtcdPath               string `json:"etcd_path"`
	CertDir                string `json:"cert_dir"`
	RunDir                 string `json:"run_dir"`
	DataDir                string `json:"data_dir"`
	HeartbeatInterval      int    `json:"heartbeat_interval_in_milliseconds"`
	ElectionTimeout        int    `json:"election_timeout_in_milliseconds"`
	PeerRequireSSL         bool   `json:"peer_require_ssl"`
	PeerIP                 string `json:"peer_ip"`
	RequireSSL             bool   `json:"require_ssl"`
	ClientIP               string `json:"client_ip"`
	DNSHealthCheckHost     string `json:"dns_health_check_host"`
	AdvertiseURLsDNSSuffix string `json:"advertise_urls_dns_suffix"`
	Machines               []string
	EnableDebugLogging     bool `json:"enable_debug_logging"`
}

type Config struct {
	Node Node
	Etcd Etcd
}

func defaultConfig() Config {
	return Config{
		Etcd: Etcd{
			EtcdPath: "/var/vcap/packages/etcd/etcd",
			CertDir:  "/var/vcap/jobs/etcd/config/certs",
			RunDir:   "/var/vcap/sys/run/etcd",
			DataDir:  "/var/vcap/store/etcd",
		},
	}
}

func ConfigFromJSONs(configFilePath, linkConfigFilePath string) (Config, error) {
	config := defaultConfig()

	configFileContents, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return Config{}, errors.New(fmt.Sprintf("error reading config file: %s", err))
	}

	if err := json.Unmarshal(configFileContents, &config); err != nil {
		return Config{}, err
	}

	linkConfigFileContents, err := ioutil.ReadFile(linkConfigFilePath)
	if err != nil {
		return Config{}, errors.New(fmt.Sprintf("error reading link config file: %s", err))
	}

	if len(linkConfigFileContents) > 0 {
		if err := json.Unmarshal(linkConfigFileContents, &config.Etcd); err != nil {
			return Config{}, err
		}
	}

	return config, nil
}

func (c Config) NodeName() string {
	return fmt.Sprintf("%s-%d", strings.Replace(c.Node.Name, "_", "-", -1), c.Node.Index)
}

func (c Config) PidFile() string {
	return filepath.Join(c.Etcd.RunDir, etcdPidFilename)
}

func (c Config) RequireSSL() bool {
	return c.Etcd.RequireSSL
}

func (c Config) CertDir() string {
	return c.Etcd.CertDir
}

func (c Config) AdvertisePeerURL() string {
	if c.Etcd.PeerRequireSSL {
		return fmt.Sprintf("https://%s.%s:%d", c.NodeName(), c.Etcd.AdvertiseURLsDNSSuffix, peerPort)
	}
	return fmt.Sprintf("http://%s:%d", c.Node.ExternalIP, peerPort)
}

func (c Config) AdvertiseClientURL() string {
	if c.Etcd.RequireSSL {
		return fmt.Sprintf("https://%s.%s:%d", c.NodeName(), c.Etcd.AdvertiseURLsDNSSuffix, clientPort)
	}
	return fmt.Sprintf("http://%s:%d", c.Node.ExternalIP, clientPort)
}

func (c Config) ListenPeerURL() string {
	protocol := "http"
	if c.Etcd.PeerRequireSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s:%d", protocol, c.Etcd.PeerIP, peerPort)
}

func (c Config) ListenClientURL() string {
	protocol := "http"
	if c.Etcd.RequireSSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s:%d", protocol, c.Etcd.ClientIP, clientPort)
}

func (c Config) EtcdClientEndpoints() []string {
	if c.Etcd.RequireSSL {
		return []string{fmt.Sprintf("https://%s:%d", c.Etcd.AdvertiseURLsDNSSuffix, clientPort)}
	} else {
		var endpoints []string
		for _, machine := range c.Etcd.Machines {
			endpoints = append(endpoints, fmt.Sprintf("http://%s:%d", machine, clientPort))
		}
		return endpoints
	}
}

func (c Config) EtcdClientSelfEndpoint() string {
	if c.Etcd.RequireSSL {
		return fmt.Sprintf("https://%s.%s:%d", c.NodeName(), c.Etcd.AdvertiseURLsDNSSuffix, clientPort)
	} else {
		return fmt.Sprintf("http://%s:%d", c.Node.ExternalIP, clientPort)
	}
}
