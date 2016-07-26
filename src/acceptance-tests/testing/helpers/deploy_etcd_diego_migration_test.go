package helpers_test

import (
	"acceptance-tests/testing/helpers"
	"io/ioutil"

	"github.com/pivotal-cf-experimental/gomegamatchers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateDiegoTLSMigrationManifest", func() {
	var (
		nonTLSDiegoManifest      string
		expectedTLSDiegoManifest string
	)

	BeforeEach(func() {
		rawContents, err := ioutil.ReadFile("fixtures/non-tls-diego-manifest.yml")
		Expect(err).NotTo(HaveOccurred())

		nonTLSDiegoManifest = string(rawContents)

		rawContents, err = ioutil.ReadFile("fixtures/tls-diego-manifest.yml")
		Expect(err).NotTo(HaveOccurred())

		expectedTLSDiegoManifest = string(rawContents)
	})

	Context("given a non-tls diego deployment", func() {
		It("generates a deployment entry with tls etcd", func() {
			tlsDiegoManifestOutput, err := helpers.CreateDiegoTLSMigrationManifest(nonTLSDiegoManifest)
			Expect(err).NotTo(HaveOccurred())
			Expect(tlsDiegoManifestOutput).To(gomegamatchers.MatchYAML(expectedTLSDiegoManifest))
		})
	})

	Context("failure cases", func() {
		It("returns an error when bad yaml is passed in", func() {
			_, err := helpers.CreateDiegoTLSMigrationManifest("%%%%%%%")
			Expect(err).To(MatchError("yaml: could not find expected directive name"))
		})
	})
})
