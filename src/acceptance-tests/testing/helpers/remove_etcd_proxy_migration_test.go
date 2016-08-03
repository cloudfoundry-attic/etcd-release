package helpers_test

import (
	"acceptance-tests/testing/helpers"
	"io/ioutil"

	"github.com/pivotal-cf-experimental/gomegamatchers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = PDescribe("RemoveEtcdProxy", func() {
	var (
		tlsManifestWithProxy    string
		tlsManifestWithoutProxy string
	)

	BeforeEach(func() {
		rawContents, err := ioutil.ReadFile("fixtures/tls-cf-manifest.yml")
		Expect(err).NotTo(HaveOccurred())

		tlsManifestWithProxy = string(rawContents)

		rawContents, err = ioutil.ReadFile("fixtures/tls-cf-manifest-without-proxy.yml")
		Expect(err).NotTo(HaveOccurred())

		tlsManifestWithoutProxy = string(rawContents)
	})

	It("returns cf manifest without etcd proxy job", func() {
		manifestWithoutProxy, err := helpers.RemoveEtcdProxy(tlsManifestWithProxy)
		Expect(err).NotTo(HaveOccurred())
		Expect(manifestWithoutProxy).To(gomegamatchers.MatchYAML(tlsManifestWithoutProxy))
	})

	Context("failure cases", func() {
		It("returns an error when bad yaml is passed in", func() {
			_, err := helpers.RemoveEtcdProxy("%%%%%%%")
			Expect(err).To(MatchError("yaml: could not find expected directive name"))
		})
	})
})
