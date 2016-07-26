package helpers

import "gopkg.in/yaml.v2"

func RemoveEtcdProxy(currentManifest string) (string, error) {
	var newManifest Manifest
	err := yaml.Unmarshal([]byte(currentManifest), &newManifest)
	if err != nil {
		return "", err
	}

	for idx, job := range newManifest.Jobs {
		if job.Name == "etcd_z1" {
			newManifest.Jobs[idx].Instances = 0
			newManifest.Jobs[idx].Networks[0].StaticIPs = &[]string{}
		}
	}

	newRawManifest, err := yaml.Marshal(newManifest)
	if err != nil {
		return "", err
	}

	return string(newRawManifest), nil
}
