package deploy_test

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("SingleInstance", func() {
	var (
		name = fmt.Sprintf("etcd-%s", generator.RandomName())
	)

	BeforeEach(func() {

		By("targeting the director")
		Expect(bosh.Command("target", config.Director).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		By("creating the release")
		Expect(bosh.Command("create", "release", "--force", "--name", name).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		By("uploading the release")
		Expect(bosh.Command("upload", "release").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		customStub := fmt.Sprintf(`---
stub:
  releases:
    etcd:
      version: latest
      name: %s
`, name)

		stubFile, err := ioutil.TempFile(os.TempDir(), "")
		Expect(err).ToNot(HaveOccurred())

		_, err = stubFile.Write([]byte(customStub))
		Expect(err).ToNot(HaveOccurred())

		bosh.GenerateAndSetDeploymentManifest(config, stubFile)
	})

	AfterEach(func() {
		By("delete deployment")
		Expect(bosh.Command("-n", "delete", "deployment", name).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		By("delete release")
		Expect(bosh.Command("-n", "delete", "release", name).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("deploys one etcd node", func() {
		By("deploying")
		Expect(bosh.Command("-n", "deploy").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
		//create etcd key
		//list etcd key
	})
})
