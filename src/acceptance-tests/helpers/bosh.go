package helpers

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cloudfoundry-incubator/candiedyaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	etcdPort = "4001"
)

type Bosh struct {
	gemfilePath string
	goPath      string
}

type manifest struct {
	Jobs       []Job      `yaml:"jobs"`
	Properties Properties `yaml:"properties"`
}

type Job struct {
	Networks []Network `yaml:"networks"`
}

type Network struct {
	Name      string   `yaml:"name"`
	StaticIps []string `yaml:"static_ips"`
}

type Properties struct {
	Etcd          Etcd          `yaml:"etcd"`
	TurbulenceApi TurbulenceApi `yaml:"turbulence_api"`
}

type Etcd struct {
	Machines []string `yaml:"machines"`
}

type TurbulenceApi struct {
	// TODO: name
	Password string `yaml:"password"`
}

func NewBosh(gemfilePath string, goPath string, config Config) Bosh {
	return Bosh{
		gemfilePath: gemfilePath,
		goPath:      goPath,
	}
}

func (bosh Bosh) Command(boshArgs ...string) *gexec.Session {
	cmd := exec.Command("bundle", append([]string{"exec", "bosh"}, boshArgs...)...)
	env := os.Environ()
	cmd.Env = append(env, fmt.Sprintf("BUNDLE_GEMFILE=%s", bosh.gemfilePath))

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	return session
}

func (bosh Bosh) GenerateAndSetDeploymentManifest(
	directorUUIDStub,
	instanceCountOverridesStub,
	persistentDiskOverridesStub,
	iaasSettingsStub,
	nameOverridesStub string,
) []string {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())

	generateDeploymentManifest := filepath.Join(bosh.goPath, "src", "acceptance-tests", "scripts", "generate_deployment_manifest")
	cmd := exec.Command(
		generateDeploymentManifest,
		directorUUIDStub,
		instanceCountOverridesStub,
		persistentDiskOverridesStub,
		iaasSettingsStub,
		nameOverridesStub,
	)

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

	_, err = tmpFile.Write(session.Out.Contents())
	Expect(err).ToNot(HaveOccurred())

	Expect(bosh.Command("deployment", tmpFile.Name()).Wait(time.Second * 10)).To(gexec.Exit(0))

	manifest := new(manifest)

	tmpFile.Close()
	tmpFile, err = os.Open(tmpFile.Name())
	Expect(err).ToNot(HaveOccurred())

	decoder := candiedyaml.NewDecoder(tmpFile)
	err = decoder.Decode(manifest)
	Expect(err).ToNot(HaveOccurred())

	etcdClientURLs := make([]string, len(manifest.Properties.Etcd.Machines))
	for index, elem := range manifest.Properties.Etcd.Machines {
		etcdClientURLs[index] = "http://" + elem + ":" + etcdPort
	}

	return etcdClientURLs
}

func (bosh Bosh) GenerateAndSetDeploymentManifestTurbulence(
	directorUUIDStub,
	instanceCountOverridesStub,
	persistentDiskOverridesStub,
	iaasSettingsStub,
	turbulenceProperties,
	nameOverridesStub string,
) string {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())

	generateDeploymentManifest := filepath.Join(bosh.goPath, "src", "acceptance-tests", "scripts", "generate_turbulence_deployment_manifest")
	cmd := exec.Command(
		generateDeploymentManifest,
		directorUUIDStub,
		instanceCountOverridesStub,
		persistentDiskOverridesStub,
		iaasSettingsStub,
		turbulenceProperties,
		nameOverridesStub,
	)

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, DEFAULT_TIMEOUT).Should(gexec.Exit(0))

	_, err = tmpFile.Write(session.Out.Contents())
	Expect(err).ToNot(HaveOccurred())

	Expect(bosh.Command("deployment", tmpFile.Name()).Wait(time.Second * 10)).To(gexec.Exit(0))

	manifest := new(manifest)

	tmpFile.Close()
	tmpFile, err = os.Open(tmpFile.Name())
	Expect(err).ToNot(HaveOccurred())

	decoder := candiedyaml.NewDecoder(tmpFile)
	err = decoder.Decode(manifest)
	Expect(err).ToNot(HaveOccurred())

	return "https://turbulence:" + manifest.Properties.TurbulenceApi.Password + "@" + manifest.Jobs[0].Networks[0].StaticIps[0] + ":8080"
}
