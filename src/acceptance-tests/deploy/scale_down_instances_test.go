package deploy_test

import (
	"acceptance-tests/helpers"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	"github.com/coreos/go-etcd/etcd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Multiple Instances", func() {
	var (
		manifest helpers.Manifest
		name     = fmt.Sprintf("etcd-%s", generator.RandomName())
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
  jobs:
    etcd_z2:
      instances: 2
`, name)

		stubFile, err := ioutil.TempFile(os.TempDir(), "")
		Expect(err).ToNot(HaveOccurred())

		_, err = stubFile.Write([]byte(customStub))
		Expect(err).ToNot(HaveOccurred())

		manifest = bosh.GenerateAndSetDeploymentManifest(config, stubFile)
	})

	AfterEach(func() {
		By("delete deployment")
		Expect(bosh.Command("-n", "delete", "deployment", name).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

		By("delete release")
		Expect(bosh.Command("-n", "delete", "release", name).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("scaling from 3 node to 1", func() {
		It("succesfully scales to multiple etcd nodes", func() {
			By("deploying")
			Expect(bosh.Command("-n", "deploy").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			Expect(len(manifest.Networks)).To(Equal(3))

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

			manifest = bosh.GenerateAndSetDeploymentManifest(config, stubFile)

			By("deploying")
			Expect(bosh.Command("-n", "deploy").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			Expect(len(manifest.Networks)).To(Equal(1))
			for index, value := range manifest.Networks {
				etcdClient := etcd.NewClient([]string{value})
				eatsKey := "eats-key" + string(index)
				eatsValue := "eats-value" + string(index)

				response, err := etcdClient.Create(eatsKey, eatsValue, 60)
				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())

				response, err = etcdClient.Get(eatsKey, false, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(response.Node.Value).To(Equal(eatsValue))
			}
		})
	})
})
