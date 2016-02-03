package deploy_test

import (
	"fmt"
	"testing"

	"acceptance-tests/testing/bosh"
	"acceptance-tests/testing/etcd"
	"acceptance-tests/testing/helpers"

	goetcd "github.com/coreos/go-etcd/etcd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	config helpers.Config
	client bosh.Client
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deploy Suite")
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

func NewEtcdClient(machines []string) etcd.Client {
	return etcd.NewClient(goetcd.NewClient(machines))
}
