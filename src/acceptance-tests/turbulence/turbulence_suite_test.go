package turbulence_test

import (
	"acceptance-tests/testing/helpers"
	"errors"
	"fmt"
	"time"

	ginkgoConfig "github.com/onsi/ginkgo/config"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	turbulenceclient "github.com/pivotal-cf-experimental/bosh-test/turbulence"
	"github.com/pivotal-cf-experimental/destiny/core"
	"github.com/pivotal-cf-experimental/destiny/iaas"
	"github.com/pivotal-cf-experimental/destiny/turbulence"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestTurbulence(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "turbulence")
}

var (
	config     helpers.Config
	boshClient bosh.Client

	turbulenceManifest turbulence.Manifest
	turbulenceClient   turbulenceclient.Client
)

var _ = BeforeSuite(func() {
	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	config, err = helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	boshClient = bosh.NewClient(bosh.Config{
		URL:              fmt.Sprintf("https://%s:25555", config.BOSH.Target),
		Username:         config.BOSH.Username,
		Password:         config.BOSH.Password,
		AllowInsecureSSL: true,
	})

	By("deploying turbulence", func() {
		info, err := boshClient.Info()
		Expect(err).NotTo(HaveOccurred())

		guid, err := helpers.NewGUID()
		Expect(err).NotTo(HaveOccurred())

		manifestConfig := turbulence.Config{
			DirectorUUID: info.UUID,
			Name:         "turbulence-etcd-" + guid,
			BOSH: turbulence.ConfigBOSH{
				Target:         config.BOSH.Target,
				Username:       config.BOSH.Username,
				Password:       config.BOSH.Password,
				DirectorCACert: config.BOSH.DirectorCACert,
			},
		}

		var iaasConfig iaas.Config
		switch info.CPI {
		case "aws_cpi":
			manifestConfig.IPRange = "10.0.16.0/24"
			awsConfig := iaas.AWSConfig{
				AccessKeyID:           config.AWS.AccessKeyID,
				SecretAccessKey:       config.AWS.SecretAccessKey,
				DefaultKeyName:        config.AWS.DefaultKeyName,
				DefaultSecurityGroups: config.AWS.DefaultSecurityGroups,
				Region:                config.AWS.Region,
				RegistryHost:          config.Registry.Host,
				RegistryPassword:      config.Registry.Password,
				RegistryPort:          config.Registry.Port,
				RegistryUsername:      config.Registry.Username,
			}
			if config.AWS.Subnet == "" {
				err = errors.New("AWSSubnet is required for AWS IAAS deployment")
				return
			}
			var cidrBlock string
			cidrPool := core.NewCIDRPool("10.0.16.0", 24, 27)
			cidrBlock, err = cidrPool.Get(ginkgoConfig.GinkgoConfig.ParallelNode)
			if err != nil {
				return
			}

			manifestConfig.IPRange = cidrBlock
			awsConfig.Subnets = []iaas.AWSConfigSubnet{{ID: config.AWS.Subnet, Range: cidrBlock, AZ: "us-east-1a"}}

			iaasConfig = awsConfig
		case "warden_cpi":
			iaasConfig = iaas.NewWardenConfig()
			manifestConfig.IPRange = "10.244.16.0/24"
		default:
			Fail("unknown infrastructure type")
		}

		turbulenceManifest, err = turbulence.NewManifest(manifestConfig, iaasConfig)
		Expect(err).NotTo(HaveOccurred())

		yaml, err := turbulenceManifest.ToYAML()
		Expect(err).NotTo(HaveOccurred())

		yaml, err = boshClient.ResolveManifestVersions(yaml)
		Expect(err).NotTo(HaveOccurred())

		turbulenceManifest, err = turbulence.FromYAML(yaml)
		Expect(err).NotTo(HaveOccurred())

		_, err = boshClient.Deploy(yaml)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() ([]bosh.VM, error) {
			return boshClient.DeploymentVMs(turbulenceManifest.Name)
		}, "1m", "10s").Should(ConsistOf([]bosh.VM{
			{Index: 0, JobName: "api", State: "running"},
		}))
	})

	By("preparing turbulence client", func() {
		turbulenceUrl := fmt.Sprintf("https://turbulence:%s@%s:8080",
			turbulenceManifest.Properties.TurbulenceAPI.Password,
			turbulenceManifest.Jobs[0].Networks[0].StaticIPs[0])

		turbulenceClient = turbulenceclient.NewClient(turbulenceUrl, 5*time.Minute, 2*time.Second)
	})
})

var _ = AfterSuite(func() {
	By("deleting the turbulence deployment", func() {
		if !CurrentGinkgoTestDescription().Failed {
			err := boshClient.DeleteDeployment(turbulenceManifest.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	})
})
