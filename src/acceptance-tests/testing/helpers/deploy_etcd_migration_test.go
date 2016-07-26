package helpers_test

import (
	"acceptance-tests/testing/helpers"
	"io/ioutil"

	"github.com/pivotal-cf-experimental/gomegamatchers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeployCFEtcdMigration", func() {
	Describe("CreateCFTLSMigrationManifest", func() {
		var (
			nonTLSCFManifest      string
			expectedTLSCFManifest string
		)

		BeforeEach(func() {
			rawContents, err := ioutil.ReadFile("fixtures/non-tls-cf-manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			nonTLSCFManifest = string(rawContents)

			rawContents, err = ioutil.ReadFile("fixtures/tls-cf-manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			expectedTLSCFManifest = string(rawContents)
		})

		Context("given a non-tls cf deployment", func() {
			It("generates a deployment entry with tls etcd", func() {
				tlsCFManifestOutput, err := helpers.CreateCFTLSMigrationManifest(nonTLSCFManifest)
				Expect(err).NotTo(HaveOccurred())
				Expect(tlsCFManifestOutput).To(gomegamatchers.MatchYAML(expectedTLSCFManifest))
			})
		})

		Context("failure cases", func() {
			It("returns an error when bad yaml is passed in", func() {
				_, err := helpers.CreateCFTLSMigrationManifest("%%%%%%%")
				Expect(err).To(MatchError("yaml: could not find expected directive name"))
			})
		})
	})

})
