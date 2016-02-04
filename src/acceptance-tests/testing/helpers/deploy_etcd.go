package helpers

import (
	"errors"
	"fmt"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny"
)

func DeployEtcdWithInstanceCount(count int, client bosh.Client, config Config) (manifest destiny.Manifest, err error) {
	guid, err := NewGUID()
	if err != nil {
		return
	}

	info, err := client.Info()
	if err != nil {
		return
	}

	manifestConfig := destiny.Config{
		DirectorUUID: info.UUID,
		Name:         fmt.Sprintf("etcd-%s", guid),
	}

	switch info.CPI {
	case "aws_cpi":
		manifestConfig.IAAS = destiny.AWS
		if config.AWS.Subnet != "" {
			manifestConfig.AWS.Subnet = config.AWS.Subnet
		} else {
			err = errors.New("AWSSubnet is required for AWS IAAS deployment")
			return
		}
	case "warden_cpi":
		manifestConfig.IAAS = destiny.Warden
	default:
		err = errors.New("unknown infrastructure type")
		return
	}

	manifest = destiny.NewEtcd(manifestConfig)

	manifest.Jobs[0], manifest.Properties = destiny.SetJobInstanceCount(manifest.Jobs[0], manifest.Networks[0], manifest.Properties, count)

	yaml, err := manifest.ToYAML()
	if err != nil {
		return
	}

	yaml, err = client.ResolveManifestVersions(yaml)
	if err != nil {
		return
	}

	manifest, err = destiny.FromYAML(yaml)
	if err != nil {
		return
	}

	err = client.Deploy(yaml)
	if err != nil {
		return
	}

	return
}
