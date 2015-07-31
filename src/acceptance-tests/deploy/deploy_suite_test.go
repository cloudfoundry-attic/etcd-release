package deploy_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"acceptance-tests/helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deploy Suite")
}

var (
	DEFAULT_TIMEOUT time.Duration = time.Minute * 5

	bosh     helpers.Bosh
	config   helpers.Config
	etcdName = fmt.Sprintf("etcd-%s", generator.RandomName())

	directorUUIDStub, nameOverridesStub *os.File
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

	// change to root directory of gopath so we can create and upload the release
	err = os.Chdir(goPath)
	Expect(err).ToNot(HaveOccurred())

	config = helpers.LoadConfig()
	bosh = helpers.NewBosh(gemfilePath, goPath, config)

	By("targeting the director")
	Expect(bosh.Command("target", config.BoshTarget).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("creating the director stub")
	session = bosh.Command("status", "--uuid").Wait(DEFAULT_TIMEOUT)
	Expect(session).To(Exit(0))
	uuid := session.Out.Contents()

	uuidStub := fmt.Sprintf(`---
director_uuid: %s
`, uuid)

	directorUUIDStub, err = ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())

	_, err = directorUUIDStub.Write([]byte(uuidStub))
	Expect(err).ToNot(HaveOccurred())

	By("creating the release")
	Expect(bosh.Command("create", "release", "--force", "--name", etcdName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("creating the name overrides stub")
	nameStub := fmt.Sprintf(`---
name_overrides:
  release_name: %s
  deployment_name: %s
`, etcdName, etcdName)

	nameOverridesStub, err = ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())

	_, err = nameOverridesStub.Write([]byte(nameStub))
	Expect(err).ToNot(HaveOccurred())

	By("uploading the release")
	Expect(bosh.Command("upload", "release").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
})

var _ = AfterSuite(func() {
	By("delete release")
	Expect(bosh.Command("-n", "delete", "release", etcdName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
})
