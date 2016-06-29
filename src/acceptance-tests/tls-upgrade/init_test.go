package tls_upgrade_test

import (
	"acceptance-tests/testing/helpers"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/bosh-test/bosh"
)

var (
	config helpers.Config
	client bosh.Client
)

func TestTLSUpgrade(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "tls-upgrade")
}

var _ = BeforeSuite(func() {
	configPath, err := helpers.ConfigPath()
	Expect(err).NotTo(HaveOccurred())

	config, err = helpers.LoadConfig(configPath)
	Expect(err).NotTo(HaveOccurred())

	client = bosh.NewClient(bosh.Config{
		URL:              fmt.Sprintf("https://%s:25555", config.BOSH.Target),
		Username:         config.BOSH.Username,
		Password:         config.BOSH.Password,
		AllowInsecureSSL: true,
	})
})
