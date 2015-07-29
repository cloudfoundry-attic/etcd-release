package turbulence_test

import (
	"acceptance-tests/helpers"
	"fmt"
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

	bosh           helpers.Bosh
	config         helpers.Config
	etcdName       = fmt.Sprintf("etcd-%s", generator.RandomName())
	turbulenceName = fmt.Sprintf("turb-%s", generator.RandomName())
)

var _ = BeforeSuite(func() {
	goEnv := os.Getenv("GOPATH")
	goPath := strings.Split(goEnv, ":")[0]

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

	if config.DEFAULT_TIMEOUT != 0 {
		DEFAULT_TIMEOUT = config.DEFAULT_TIMEOUT
	}

	By("targeting the director")
	Expect(bosh.Command("target", config.Director).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("creating the release")
	Expect(bosh.Command("create", "release", "--force", "--name", etcdName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("uploading the etcd release")
	Expect(bosh.Command("upload", "release").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	// change to root directory of turbulence and upload the turbulence release
	err = os.Chdir(filepath.Join(goPath, "src", "github.com", "cppforlife", "turbulence-release"))
	Expect(err).ToNot(HaveOccurred())

	By("creating the turbulence release")
	Expect(bosh.Command("create", "release", "--name", turbulenceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("uploading the turbulence release")
	Expect(bosh.Command("upload", "release").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	err = os.Chdir(goPath)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("delete etcd release")
	Expect(bosh.Command("-n", "delete", "release", etcdName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("delete turbulence release")
	Expect(bosh.Command("-n", "delete", "release", turbulenceName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
})
