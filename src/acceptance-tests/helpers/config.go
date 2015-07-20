package helpers

import (
	"encoding/json"
	"os"
)

type Config struct {
	Director string `json:"director"`
	Stub     string `json:"stub"` // Location of stub to use for testing
	/*
	*		For fresh deploy, we write release and generate
	*		For scale, we write job instance count and generate
	 */
}

var loadedConfig *Config

func LoadConfig() Config {
	if loadedConfig == nil {
		loadedConfig = loadConfigJsonFromPath()
	}

	if loadedConfig.Director == "" {
		panic("missing director endpoint")
	}

	if loadedConfig.Stub == "" {
		panic("missing stub path")
	}

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
	path := os.Getenv("CONFIG")
	if path == "" {
		panic("Must set $CONFIG to point to an integration config .json file.")
	}

	return path
}
