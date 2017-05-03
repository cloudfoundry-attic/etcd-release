package config

import (
	"encoding/json"
	"io/ioutil"
)

type Node struct {
	Name       string
	Index      int
	ExternalIP string `json:"external_ip"`
}

type Etcd struct {
	HeartbeatInterval      int    `json:"heartbeat_interval_in_milliseconds"`
	ElectionTimeout        int    `json:"election_timeout_in_milliseconds"`
	PeerRequireSSL         bool   `json:"peer_require_ssl"`
	PeerIP                 string `json:"peer_ip"`
	RequireSSL             bool   `json:"require_ssl"`
	ClientIP               string `json:"client_ip"`
	AdvertiseURLsDNSSuffix string `json:"advertise_urls_dns_suffix"`
}

type Config struct {
	Node Node
	Etcd Etcd
}

func ConfigFromJSON(configFilePath string) (Config, error) {
	configFileContents, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := json.Unmarshal(configFileContents, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}
