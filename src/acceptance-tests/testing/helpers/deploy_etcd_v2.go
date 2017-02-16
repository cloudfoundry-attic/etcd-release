package helpers

import (
	"fmt"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/etcd"
)

func DeployEtcdV2WithInstanceCount(deploymentPrefix string, count int, client bosh.Client, config Config) (string, error) {
	manifest, err := NewEtcdV2WithInstanceCount(deploymentPrefix, count, client, config)
	if err != nil {
		return "", err
	}

	err = ResolveVersionsAndDeployV2(manifest, client)
	if err != nil {
		return "", err
	}

	return manifest, nil
}

func NewEtcdV2WithInstanceCount(deploymentPrefix string, count int, client bosh.Client, config Config) (string, error) {
	guid, err := NewGUID()
	if err != nil {
		return "", err
	}

	info, err := client.Info()
	if err != nil {
		return "", err
	}

	manifestConfig := etcd.ConfigV2{
		DirectorUUID: info.UUID,
		Name:         fmt.Sprintf("etcd-%s-%s", deploymentPrefix, guid),
		AZs:          []string{"z3", "z4"},
	}

	manifest, err := etcd.NewManifestV2(manifestConfig)
	if err != nil {
		return "", err
	}

	return manifest, nil
}
