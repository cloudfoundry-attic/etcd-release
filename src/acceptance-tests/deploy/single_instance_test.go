package deploy_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"acceptance-tests/helpers"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var DEFAULT_TIMEOUT = 2 * time.Minute

func boshCommand(boshArgs ...string) {
	cmd := exec.Command("bundle", append([]string{"exec", "bosh"}, boshArgs...)...)
	env := os.Environ()
	cmd.Env = append(env, fmt.Sprintf("BUNDLE_GEMFILE=%s", gemfilePath))

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))
}

func changeRootDirectory() {
	err := os.Chdir(suitePath)

	Expect(err).ToNot(HaveOccurred())
}

func generateManifest(config helpers.Config, customStub *os.File) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())

	generateDeploymentManifest := filepath.Join(suitePath, "generate_deployment_manifest")
	cmd := exec.Command(generateDeploymentManifest, config.Stub, customStub.Name())

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

	tmpFile.Write(session.Out.Contents())

	boshCommand("deployment", tmpFile.Name())
}

var _ = Describe("SingleInstance", func() {
	var (
		name string
	)

	BeforeEach(func() {
		name = generator.RandomName()

		changeRootDirectory()
		config := helpers.LoadConfig()

		By("targeting the director")
		boshCommand("target", config.Director)

		By("creating the release")
		boshCommand("create", "release", "--force", "--name", name)

		By("uploading the release")
		boshCommand("upload", "release")

		// By("setting the deployment")

		customStub := fmt.Sprintf(`---
stub:
  releases:
    etcd:
      version: latest
      name: %s
`, name)

		stubFile, err := ioutil.TempFile(os.TempDir(), "")
		Expect(err).ToNot(HaveOccurred())

		stubFile.Write([]byte(customStub))

		generateManifest(config, stubFile)
	})

	AfterEach(func() {
		//delete deployment
		By("delete deployment")
		boshCommand("-n", "delete", "deployment", name)

		//delete release
		By("delete release")
		boshCommand("-n", "delete", "release", name)
	})

	It("deploys one etcd node", func() {
		By("deploying")
		boshCommand("-n", "deploy")
		//create etcd key
		//list etcd key
	})
})
