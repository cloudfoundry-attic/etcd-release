package helpers

import (
	"encoding/json"
	"os"
	"time"
)

type Config struct {
	BoshTarget                     string `json:"bosh_target"`
	IAASSettingsEtcdStubPath       string `json:"iaas_settings_etcd_stub_path"`
	IAASSettingsTurbulenceStubPath string `json:"iaas_settings_turbulence_stub_path"`
	CPIReleaseUrl                  string `json:"cpi_release_url"`
	CPIReleaseName                 string `json:"cpi_release_name"`
	DefaultTimeoutString           string `json:"default_timeout"`
	DefaultTimeout                 time.Duration
}

var loadedConfig *Config

func LoadConfig() Config {
	if loadedConfig == nil {
		loadedConfig = loadConfigJsonFromPath()
	}

	if loadedConfig.BoshTarget == "" {
		panic("missing BOSH target (e.g. 'lite' or '192.168.50.4'")
	}

	if loadedConfig.IAASSettingsEtcdStubPath == "" {
		panic("missing etcd IaaS settings stub path")
	}

	if loadedConfig.DefaultTimeoutString == "" {
		loadedConfig.DefaultTimeoutString = "5m"
	}

	duration, err := time.ParseDuration(loadedConfig.DefaultTimeoutString)
	if err != nil {
		panic("invalid timeout string")
	}

	loadedConfig.DefaultTimeout = duration

	return *loadedConfig
}

func loadConfigJsonFromPath() *Config {
	var config *Config = &Config{}

	path := configPath()

	configFile, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(config)
	if err != nil {
		panic(err)
	}

	return config
}

func configPath() string {
	path := os.Getenv("EATS_CONFIG")
	if path == "" {
		panic("Must set $EATS_CONFIG to point to an etcd acceptance tests config file.")
	}

	return path
}
