package turbulence_test

import (
	"acceptance-tests/testing/helpers"
	"fmt"

	"github.com/pivotal-cf-experimental/bosh-test/bosh"
	turbulenceclient "github.com/pivotal-cf-experimental/bosh-test/turbulence"
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
})
