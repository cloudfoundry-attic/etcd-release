package deploy_test

import (
	"acceptance-tests/helpers"
	"fmt"

	"github.com/coreos/go-etcd/etcd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Single Instance Rolling deploys", func() {
	var (
		etcdManifest   = new(helpers.Manifest)
		etcdClientURLs []string
	)

	BeforeEach(func() {
		bosh.GenerateAndSetDeploymentManifest(
			etcdManifest,
			etcdManifestGeneration,
			directorUUIDStub,
			helpers.InstanceCount1NodeStubPath,
			helpers.PersistentDiskStubPath,
			config.IAASSettingsEtcdStubPath,
			helpers.PropertyOverridesStubPath,
			etcdNameOverrideStub,
		)

		for _, elem := range etcdManifest.Properties.Etcd.Machines {
			etcdClientURLs = append(etcdClientURLs, "http://"+elem+":4001")
		}

		By("deploying")
		Expect(bosh.Command("-n", "deploy").Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))
		Expect(len(etcdManifest.Properties.Etcd.Machines)).To(Equal(1))
	})

	AfterEach(func() {
		By("delete deployment")
		Expect(bosh.Command("-n", "delete", "deployment", etcdDeployment).Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))
	})

	It("Saves data after a rolling deploy", func() {
		By("setting a persistent value")
		etcdClient := etcd.NewClient(etcdClientURLs)

		eatsKey := "eats-key"
		eatsValue := "eats-value"

		response, err := etcdClient.Create(eatsKey, eatsValue, 6000)
		Expect(err).ToNot(HaveOccurred())
		Expect(response).ToNot(BeNil())

		// generate new stub that overwrites a property
		etcdStub := fmt.Sprintf(`---
property_overrides:
  etcd:
    heartbeat_interval_in_milliseconds: 51
`)

		etcdRollingDeployStub := helpers.WriteStub(etcdStub)

		bosh.GenerateAndSetDeploymentManifest(
			etcdManifest,
			etcdManifestGeneration,
			directorUUIDStub,
			helpers.InstanceCount1NodeStubPath,
			helpers.PersistentDiskStubPath,
			config.IAASSettingsEtcdStubPath,
			etcdRollingDeployStub,
			etcdNameOverrideStub,
		)

		By("deploying")
		Expect(bosh.Command("-n", "deploy").Wait(helpers.DEFAULT_TIMEOUT)).To(Exit(0))

		By("reading each value from each machine")
		for _, url := range etcdClientURLs {
			etcdClient := etcd.NewClient([]string{url})

			response, err := etcdClient.Get(eatsKey, false, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Node.Value).To(Equal(eatsValue))
		}
	})
})
