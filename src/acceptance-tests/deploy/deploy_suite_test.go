package deploy_test

import (
	"fmt"
	"testing"

	"acceptance-tests/testing/helpers"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	"github.com/pivotal-cf-experimental/destiny/turbulence"

	turbulenceclient "github.com/pivotal-cf-experimental/bosh-test/turbulence"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	config             helpers.Config
	client             bosh.Client
	turbulenceManifest turbulence.Manifest
	turbulenceClient   turbulenceclient.Client
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "deploy")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	config, client = bootstrapSuite()

	turbulenceManifest, err := helpers.DeployTurbulence(client, config)
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() ([]bosh.VM, error) {
		return helpers.DeploymentVMs(client, turbulenceManifest.Name)
	}, "1m", "10s").Should(ConsistOf(helpers.GetTurbulenceVMsFromManifest(turbulenceManifest)))

	turbulenceManifestBytes, err := turbulenceManifest.ToYAML()
	Expect(err).NotTo(HaveOccurred())

	return turbulenceManifestBytes
}, func(turbulenceManifestBytes []byte) {
	var err error
	turbulenceManifest, err = turbulence.FromYAML(turbulenceManifestBytes)
	Expect(err).NotTo(HaveOccurred())

	config, client = bootstrapSuite()
	turbulenceClient = helpers.NewTurbulenceClient(turbulenceManifest)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	if len(turbulenceManifest.Name) != 0 {
		err := client.DeleteDeployment(turbulenceManifest.Name)
		Expect(err).NotTo(HaveOccurred())
	}
})

func bootstrapSuite() (helpers.Config, bosh.Client) {
	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	config, err := helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	client := bosh.NewClient(bosh.Config{
		URL:              fmt.Sprintf("https://%s:25555", config.BOSH.Target),
		Username:         config.BOSH.Username,
		Password:         config.BOSH.Password,
		AllowInsecureSSL: true,
	})

	return config, client
}
