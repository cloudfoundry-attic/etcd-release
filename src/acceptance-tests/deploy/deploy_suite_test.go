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
	config helpers.Config

	bosh *helpers.Bosh

	etcdManifestGeneration string

	directorUUIDStub string

	etcdRelease          = fmt.Sprintf("etcd-%s", generator.RandomName())
	etcdDeployment       = etcdRelease
	etcdNameOverrideStub string
)

var _ = BeforeSuite(func() {
	goPath = helpers.SetupGoPath()
	gemfilePath := helpers.SetupFastBosh()
	config = helpers.LoadConfig()
	boshOperationTimeout := helpers.GetBoshOperationTimeout(config)
	bosh = helpers.NewBosh(gemfilePath, goPath, config.BoshTarget, boshOperationTimeout)

	etcdManifestGeneration = filepath.Join(goPath, "src", "acceptance-tests", "scripts", "generate_etcd_deployment_manifest")

	err := os.Chdir(goPath)
	Expect(err).ToNot(HaveOccurred())

	directorUUIDStub = bosh.TargetDeployment()
	createEtcdStub()
	bosh.CreateAndUploadRelease(goPath, etcdRelease)
})

var _ = AfterSuite(func() {
	if bosh == nil {
		return
	}

	By("delete release")
	bosh.Command("-n", "delete", "release", etcdRelease)
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
