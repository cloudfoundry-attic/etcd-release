package deploy_test

import (
	"acceptance-tests/helpers"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/coreos/go-etcd/etcd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Multiple Instances", func() {
	var (
		manifest helpers.Manifest
	)

	BeforeEach(func() {
		customStub := fmt.Sprintf(`---
stub:
  releases:
    etcd:
      version: latest
      name: %s
  jobs:
    etcd_z1:
      instances: 1
    etcd_z2:
      instances: 0
`, etcdName)

		stubFile, err := ioutil.TempFile(os.TempDir(), "")
		Expect(err).ToNot(HaveOccurred())

		_, err = stubFile.Write([]byte(customStub))
		Expect(err).ToNot(HaveOccurred())

		manifest = bosh.GenerateAndSetDeploymentManifest(config, stubFile)
	})

	AfterEach(func() {
		By("delete deployment")
		Expect(bosh.Command("-n", "delete", "deployment", etcdName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))
	})

	Describe("scaling from 1 node to 3", func() {
		It("succesfully scales to multiple etcd nodes", func() {
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

			customStub := fmt.Sprintf(`---
stub:
  releases:
    etcd:
      version: latest
      name: %s
  jobs:
    etcd_z1:
      instances: 1
    etcd_z2:
      instances: 2
`, etcdName)

			stubFile, err := ioutil.TempFile(os.TempDir(), "")
			Expect(err).ToNot(HaveOccurred())

			_, err = stubFile.Write([]byte(customStub))
			Expect(err).ToNot(HaveOccurred())

			manifest = bosh.GenerateAndSetDeploymentManifest(config, stubFile)

			By("deploying")
			Expect(bosh.Command("-n", "deploy").Wait(DEFAULT_TIMEOUT)).To(Exit(0))

			Expect(len(manifest.Networks)).To(Equal(3))
			for index, value := range manifest.Networks {
				etcdClient := etcd.NewClient([]string{value})

				eatsKey := fmt.Sprintf("eats-key%d", index)
				eatsValue := fmt.Sprintf("eats-value%d", index)

				response, err := etcdClient.Create(eatsKey, eatsValue, 60)
				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())
			}

			for _, value := range manifest.Networks {
				etcdClient := etcd.NewClient([]string{value})

				for index, _ := range manifest.Networks {
					eatsKey := fmt.Sprintf("eats-key%d", index)
					eatsValue := fmt.Sprintf("eats-value%d", index)

					response, err := etcdClient.Get(eatsKey, false, false)
					Expect(err).ToNot(HaveOccurred())
					Expect(response.Node.Value).To(Equal(eatsValue))
				}
			}
		})
	})
})
