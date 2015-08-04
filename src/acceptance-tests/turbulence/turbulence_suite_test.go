package turbulence_test

import (
	"acceptance-tests/helpers"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"testing"
	"time"
)

func TestTurbulence(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Turbulence Suite")
}

var (
	DEFAULT_TIMEOUT time.Duration = time.Minute * 5

	goPath         string
	bosh           helpers.Bosh
	config         helpers.Config
	turbulencUrl   string
	etcdName       = fmt.Sprintf("etcd-%s", generator.RandomName())
	turbulenceName = fmt.Sprintf("turb-%s", generator.RandomName())

	directorUUIDStub, etcdNameOverrideStub, turbulenceNameOverrideStub *os.File
)

var _ = BeforeSuite(func() {
	goEnv := os.Getenv("GOPATH")
	goPath = strings.Split(goEnv, ":")[0]

	// setup fast bosh when running locally
	wd, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	gemfilePath := filepath.Join(wd, "..", "Gemfile")

	cmd := exec.Command("bundle")
	env := os.Environ()
	cmd.Env = append(env, fmt.Sprintf("BUNDLE_GEMFILE=%s", gemfilePath))

	session, err := Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, time.Minute*5).Should(Exit(0))

	// change to root directory of gopath so we can create and upload the etcd release
	err = os.Chdir(goPath)
	Expect(err).ToNot(HaveOccurred())

	config = helpers.LoadConfig()
	bosh = helpers.NewBosh(gemfilePath, goPath, config)

	targetDeployment()

	uploadEtcd()

	uploadBoshCpiRelease()

	uploadAndDeployTurbulence()
})

func targetDeployment() {
	By("targeting the director")
	Expect(bosh.Command("target", config.BoshTarget).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("creating the director stub")
	session := bosh.Command("status", "--uuid").Wait(DEFAULT_TIMEOUT)
	Expect(session).To(Exit(0))
	uuid := session.Out.Contents()

	uuidStub := fmt.Sprintf(`---
director_uuid: %s
`, uuid)

	var err error
	directorUUIDStub, err = ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())
	defer directorUUIDStub.Close()

	_, err = directorUUIDStub.Write([]byte(uuidStub))
	Expect(err).ToNot(HaveOccurred())
}

func uploadEtcd() {
	nameStub := fmt.Sprintf(`---
name_overrides:
  release_name: %s
  deployment_name: %s
`, etcdName, etcdName)

	var err error
	etcdNameOverrideStub, err = ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())
	defer etcdNameOverrideStub.Close()

	_, err = etcdNameOverrideStub.Write([]byte(nameStub))
	Expect(err).ToNot(HaveOccurred())

	By("creating the etcd release")
	Expect(bosh.Command("create", "release", "--force", "--name", etcdName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("uploading the etcd release")
	Expect(bosh.Command("upload", "release").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func uploadAndDeployTurbulence() {
	err := os.Chdir(filepath.Join(goPath, "src", "github.com", "cppforlife", "turbulence-release"))
	Expect(err).ToNot(HaveOccurred())

	By("creating the turbulence release")
	Expect(bosh.Command("create", "release", "--name", turbulenceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("uploading the turbulence release")
	Expect(bosh.Command("upload", "release").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	err = os.Chdir(goPath)
	Expect(err).ToNot(HaveOccurred())

	By("creating the turbulence overrides stub")
	nameStub := fmt.Sprintf(`---
name_overrides:
  deployment_name: %s
  turbulence:
    release_name: %s
  cpi:
    release_name: %s
`, turbulenceName, turbulenceName, config.CPIReleaseName)

	turbulenceNameOverrideStub, err = ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())

	_, err = turbulenceNameOverrideStub.Write([]byte(nameStub))
	Expect(err).ToNot(HaveOccurred())
	turbulenceNameOverrideStub.Close()

	turbulencUrl = bosh.GenerateAndSetDeploymentManifestTurbulence(
		directorUUIDStub.Name(),
		helpers.TurbulenceInstanceCountOverridesStubPath,
		helpers.TurbulencePersistentDiskOverridesStubPath,
		config.IAASSettingsTurbulenceStubPath,
		helpers.TurbulencePropertiesStubPath,
		turbulenceNameOverrideStub.Name(),
	)

	By("deploying the turbulence release")
	Expect(bosh.Command("-n", "deploy").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func uploadBoshCpiRelease() {
	if config.CPIReleaseUrl == "" {
		panic("missing required cpi release url")
	}

	if config.CPIReleaseName == "" {
		panic("missing required cpi release name")
	}

	Expect(bosh.Command("-n", "upload", "release", config.CPIReleaseUrl, "--skip-if-exists").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

var _ = AfterSuite(func() {
	By("delete etcd release")
	Expect(bosh.Command("-n", "delete", "release", etcdName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("delete turbulence release")
	Expect(bosh.Command("-n", "delete", "deployment", turbulenceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("delete turbulence release")
	Expect(bosh.Command("-n", "delete", "release", turbulenceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("deleting the cpi release")
	Expect(bosh.Command("-n", "delete", "release", config.CPIReleaseName))
})
