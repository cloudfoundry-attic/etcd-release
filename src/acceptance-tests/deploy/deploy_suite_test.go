package deploy_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"acceptance-tests/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"

	"testing"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deploy Suite")
}

var (
	DEFAULT_TIMEOUT time.Duration = time.Minute * 5

	bosh   helpers.Bosh
	config helpers.Config
	name = fmt.Sprintf("etcd-%s", generator.RandomName())
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
	Eventually(session).Should(Exit(0))

	// change to root directory of gopath so we can create and upload the release
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
	Expect(bosh.Command("create", "release", "--force", "--name", name).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("uploading the release")
	Expect(bosh.Command("upload", "release").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
})

var _ =	AfterSuite(func() {
	By("delete deployment")
	Expect(bosh.Command("-n", "delete", "deployment", name).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("delete release")
	Expect(bosh.Command("-n", "delete", "release", name).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
})
