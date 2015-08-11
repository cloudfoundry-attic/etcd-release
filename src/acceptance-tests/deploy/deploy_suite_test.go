package deploy_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"acceptance-tests/helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deploy Suite")
}

var (
	goPath string
	bosh   helpers.Bosh
	config helpers.Config

	etcdRelease    = fmt.Sprintf("etcd-%s", generator.RandomName())
	etcdDeployment = etcdRelease

	directorUUIDStub, etcdNameOverrideStub, turbulenceNameOverrideStub string

	etcdManifestGeneration string
)

var _ = BeforeSuite(func() {
	goPath = helpers.SetupGoPath()
	gemfilePath := helpers.SetupFastBosh()
	config = helpers.LoadConfig()
	bosh = helpers.NewBosh(gemfilePath, goPath, config.BoshTarget)

	etcdManifestGeneration = filepath.Join(goPath, "src", "acceptance-tests", "scripts", "generate_etcd_deployment_manifest")

	err := os.Chdir(goPath)
	Expect(err).ToNot(HaveOccurred())

	directorUUIDStub = bosh.TargetDeployment()
	createEtcdStub()
	bosh.CreateAndUploadRelease(goPath, etcdRelease)
})

var _ = AfterSuite(func() {
	By("delete release")
	bosh.Command("-n", "delete", "release", etcdRelease).Wait(config.DefaultTimeout)
})

func createEtcdStub() {
	By("creating the etcd overrides stub")
	etcdStub := fmt.Sprintf(`---
name_overrides:
  release_name: %s
  deployment_name: %s
`, etcdRelease, etcdDeployment)

	etcdNameOverrideStub = helpers.WriteStub(etcdStub)
}
