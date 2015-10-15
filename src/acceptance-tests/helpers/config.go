package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	BoshTarget                     string `json:"bosh_target"`
	IAASSettingsEtcdStubPath       string `json:"iaas_settings_etcd_stub_path"`
	IAASSettingsTurbulenceStubPath string `json:"iaas_settings_turbulence_stub_path"`
	TurbulencePropertiesStubPath   string `json:"turbulence_properties_stub_path"`
	CPIReleaseUrl                  string `json:"cpi_release_url"`
	CPIReleaseName                 string `json:"cpi_release_name"`
	BoshOperationTimeout           string `json:"bosh_operation_timeout"`
	TurbulenceOperationTimeout     string `json:"turbulence_operation_timeout"`
	TurbulenceReleaseUrl           string `json:"turbulence_release_url"`
	TurbulenceReleaseName          string
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

	loadedConfig.TurbulenceReleaseName = "turbulence"

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

func GetBoshOperationTimeout(config Config) time.Duration {
	if config.BoshOperationTimeout == "" {
		return defaultBoshOperationTimeout
	}

	duration, err := time.ParseDuration(config.BoshOperationTimeout)
	if err != nil {
		panic(fmt.Sprintf("invalid duration string for BOSH operation timeout config: '%s'", config.BoshOperationTimeout))
	}

	return duration
}

func GetTurbulenceOperationTimeout(config Config) time.Duration {
	if config.TurbulenceOperationTimeout == "" {
		return defaultTurbulenceOperationTimeout
	}

	duration, err := time.ParseDuration(config.TurbulenceOperationTimeout)
	if err != nil {
		panic(fmt.Sprintf("invalid duration string for Turbulence operation timeout config: '%s'", config.TurbulenceOperationTimeout))
	}

	return duration
}
