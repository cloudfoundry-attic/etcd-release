package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
)

type Node struct {
	Name       string
	Index      int
	ExternalIP string `json:"external_ip"`
}

type Etcd struct {
	EtcdPath               string `json:"etcd_path"`
	HeartbeatInterval      int    `json:"heartbeat_interval_in_milliseconds"`
	ElectionTimeout        int    `json:"election_timeout_in_milliseconds"`
	PeerRequireSSL         bool   `json:"peer_require_ssl"`
	PeerIP                 string `json:"peer_ip"`
	RequireSSL             bool   `json:"require_ssl"`
	ClientIP               string `json:"client_ip"`
	AdvertiseURLsDNSSuffix string `json:"advertise_urls_dns_suffix"`
	Machines               []string
}

type Config struct {
	Node Node
	Etcd Etcd
}

func ConfigFromJSONs(configFilePath, linkConfigFilePath string) (Config, error) {
	configFileContents, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return Config{}, errors.New(fmt.Sprintf("error reading config file: %s", err))
	}

	config := Config{
		Etcd: Etcd{
			EtcdPath: "/var/vcap/packages/etcd/etcd",
		},
	}

	if err := json.Unmarshal(configFileContents, &config); err != nil {
		return Config{}, err
	}

	linkConfigFileContents, err := ioutil.ReadFile(linkConfigFilePath)
	if err != nil {
		return Config{}, errors.New(fmt.Sprintf("error reading link config file: %s", err))
	}

	if err := json.Unmarshal(linkConfigFileContents, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}

func (c Config) NodeName() string {
	return fmt.Sprintf("%s-%d", strings.Replace(c.Node.Name, "_", "-", -1), c.Node.Index)
}

func (c Config) AdvertisePeerURL() string {
	if c.Etcd.PeerRequireSSL {
		return fmt.Sprintf("https://%s.%s:7001", c.NodeName(), c.Etcd.AdvertiseURLsDNSSuffix)
	}
	return fmt.Sprintf("http://%s:7001", c.Node.ExternalIP)
}
