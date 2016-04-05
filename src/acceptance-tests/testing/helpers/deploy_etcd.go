package helpers

import (
	"errors"
	"fmt"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/etcd"
	"github.com/pivotal-cf-experimental/destiny/iaas"
)

func DeployEtcdWithInstanceCount(count int, client bosh.Client, config Config, enableSSL bool) (manifest etcd.Manifest, err error) {
	guid, err := NewGUID()
	if err != nil {
		return
	}

	info, err := client.Info()
	if err != nil {
		return
	}

	manifestConfig := etcd.Config{
		DirectorUUID: info.UUID,
		Name:         fmt.Sprintf("etcd-%s", guid),
	}

	var iaasConfig iaas.Config
	switch info.CPI {
	case "aws_cpi":
		iaasConfig = iaas.AWSConfig{
			AccessKeyID:           config.AWS.AccessKeyID,
			SecretAccessKey:       config.AWS.SecretAccessKey,
			DefaultKeyName:        config.AWS.DefaultKeyName,
			DefaultSecurityGroups: config.AWS.DefaultSecurityGroups,
			Region:                config.AWS.Region,
			Subnet:                config.AWS.Subnet,
			RegistryHost:          config.Registry.Host,
			RegistryPassword:      config.Registry.Password,
			RegistryPort:          config.Registry.Port,
			RegistryUsername:      config.Registry.Username,
		}
		if config.AWS.Subnet != "" {
			manifestConfig.IPRange = "10.0.16.0/24"
		} else {
			err = errors.New("AWSSubnet is required for AWS IAAS deployment")
			return
		}
	case "warden_cpi":
		iaasConfig = iaas.NewWardenConfig()
		manifestConfig.IPRange = "10.244.16.0/24"
	default:
		err = errors.New("unknown infrastructure type")
		return
	}

	if enableSSL {
		manifest = etcd.NewTLSManifest(manifestConfig, iaasConfig)
	} else {
		manifest = etcd.NewManifest(manifestConfig, iaasConfig)
	}

	manifest.Jobs[1], manifest.Properties = etcd.SetJobInstanceCount(manifest.Jobs[1], manifest.Networks[0], manifest.Properties, count)

	yaml, err := manifest.ToYAML()
	if err != nil {
		return
	}

	yaml, err = client.ResolveManifestVersions(yaml)
	if err != nil {
		return
	}

	manifest, err = etcd.FromYAML(yaml)
	if err != nil {
		return
	}

	_, err = client.Deploy(yaml)
	if err != nil {
		return
	}

	return
}
