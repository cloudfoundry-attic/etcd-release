package turbulence_test

import (
	"acceptance-tests/helpers"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"testing"
)

func TestTurbulence(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Turbulence Suite")
}

var (
	goPath             string
	turbulenceUrl      string
	bosh               helpers.Bosh
	config             helpers.Config
	turbulenceManifest *helpers.Manifest

	etcdRelease          = fmt.Sprintf("etcd-%s", generator.RandomName())
	etcdDeployment       = etcdRelease
	turbulenceRelease    = fmt.Sprintf("turb-%s", generator.RandomName())
	turbulenceDeployment = turbulenceRelease

	directorUUIDStub, etcdNameOverrideStub, turbulenceNameOverrideStub string

	turbulenceManifestGeneration string
	etcdManifestGeneration       string
)

var _ = BeforeSuite(func() {
	goPath = helpers.SetupGoPath()
	gemfilePath := helpers.SetupFastBosh()
	config = helpers.LoadConfig()
	bosh = helpers.NewBosh(gemfilePath, goPath, config.BoshTarget)

	turbulenceManifestGeneration = filepath.Join(goPath, "src", "acceptance-tests", "scripts", "generate_turbulence_deployment_manifest")
	etcdManifestGeneration = filepath.Join(goPath, "src", "acceptance-tests", "scripts", "generate_etcd_deployment_manifest")

	directorUUIDStub = bosh.TargetDeployment()

	err := os.Chdir(goPath)
	Expect(err).ToNot(HaveOccurred())

	uploadBoshCpiRelease()

	createTurbulenceStub()

	turbulenceManifest = new(helpers.Manifest)
	bosh.GenerateAndSetDeploymentManifest(
		turbulenceManifest,
		turbulenceManifestGeneration,
		directorUUIDStub,
		helpers.TurbulenceInstanceCountOverridesStubPath,
		helpers.TurbulencePersistentDiskOverridesStubPath,
		config.IAASSettingsTurbulenceStubPath,
		helpers.TurbulencePropertiesStubPath,
		turbulenceNameOverrideStub,
	)

	bosh.CreateUploadAndDeployRelease(
		filepath.Join(goPath, "src", "github.com", "cppforlife", "turbulence-release"),
		turbulenceRelease,
		turbulenceDeployment)

	createEtcdStub()
	bosh.CreateAndUploadRelease(goPath, etcdRelease)
})

var _ = AfterSuite(func() {
	By("delete etcd release")
	Expect(bosh.Command("-n", "delete", "release", etcdRelease).Wait(config.DefaultTimeout)).To(Exit(0))

	By("delete turbulence deployment")
	Expect(bosh.Command("-n", "delete", "deployment", turbulenceDeployment).Wait(config.DefaultTimeout)).To(Exit(0))

	By("delete turbulence release")
	Expect(bosh.Command("-n", "delete", "release", turbulenceRelease).Wait(config.DefaultTimeout)).To(Exit(0))

	By("deleting the cpi release")
	Expect(bosh.Command("-n", "delete", "release", config.CPIReleaseName))
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

func createTurbulenceStub() {
	By("creating the turbulence overrides stub")
	turbulenceStub := fmt.Sprintf(`---
name_overrides:
  deployment_name: %s
  turbulence:
    release_name: %s
  cpi:
    release_name: %s
`, turbulenceDeployment, turbulenceRelease, config.CPIReleaseName)

	turbulenceNameOverrideStub = helpers.WriteStub(turbulenceStub)
}

func uploadBoshCpiRelease() {
	if config.CPIReleaseUrl == "" {
		panic("missing required cpi release url")
	}

	if config.CPIReleaseName == "" {
		panic("missing required cpi release name")
	}

	Expect(bosh.Command("-n", "upload", "release", config.CPIReleaseUrl, "--skip-if-exists").Wait(config.DefaultTimeout)).To(Exit(0))
}
