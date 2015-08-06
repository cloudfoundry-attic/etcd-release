package helpers

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"github.com/cloudfoundry-incubator/candiedyaml"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

type Bosh struct {
	gemfilePath string
	goPath      string
	target      string
}

type Manifest struct {
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

func NewBosh(gemfilePath string, goPath string, target string) Bosh {
	return Bosh{
		gemfilePath: gemfilePath,
		goPath:      goPath,
		target:      target,
	}
}

func WriteStub(stub string) string {
	stubFile, err := ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())

	defer stubFile.Close()

	_, err = stubFile.Write([]byte(stub))
	Expect(err).ToNot(HaveOccurred())

	return stubFile.Name()
}

func (bosh Bosh) TargetDeployment() string {
	By("targeting the director")
	Expect(bosh.Command("target", bosh.target).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("creating the director stub")
	session := bosh.Command("status", "--uuid").Wait(DEFAULT_TIMEOUT)
	Expect(session).To(Exit(0))
	uuid := session.Out.Contents()

	uuidStub := fmt.Sprintf(`---
director_uuid: %s
`, uuid)

	var err error
	directorUUIDStub, err := ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())
	defer directorUUIDStub.Close()

	_, err = directorUUIDStub.Write([]byte(uuidStub))
	Expect(err).ToNot(HaveOccurred())

	return directorUUIDStub.Name()
}

func (bosh Bosh) CreateAndUploadRelease(releaseDir, releaseName string) {
	err := os.Chdir(releaseDir)
	Expect(err).ToNot(HaveOccurred())

	By("creating the etcd release")
	Expect(bosh.Command("create", "release", "--force", "--name", releaseName).Wait(DEFAULT_TIMEOUT)).To(Exit(0))

	By("uploading the etcd release")
	Expect(bosh.Command("upload", "release").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func (bosh Bosh) CreateUploadAndDeployRelease(releaseDir, releaseName, deploymentName string) {
	bosh.CreateAndUploadRelease(releaseDir, releaseName)

	By("deploying the turbulence release")
	Expect(bosh.Command("-n", "deploy").Wait(DEFAULT_TIMEOUT)).To(Exit(0))
}

func (bosh Bosh) Command(boshArgs ...string) *Session {
	cmd := exec.Command("bundle", append([]string{"exec", "bosh"}, boshArgs...)...)
	env := os.Environ()
	cmd.Env = append(env, fmt.Sprintf("BUNDLE_GEMFILE=%s", bosh.gemfilePath))

	session, err := Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())

	return session
}

func (bosh Bosh) GenerateAndSetDeploymentManifest(manifest interface{}, manifestGenerateScripts string, stubs ...string) {
	cmd := exec.Command(manifestGenerateScripts, stubs...)
	session, err := Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session, DEFAULT_TIMEOUT).Should(Exit(0))

	tmpFile, err := ioutil.TempFile(os.TempDir(), "")
	Expect(err).ToNot(HaveOccurred())
	_, err = tmpFile.Write(session.Out.Contents())
	Expect(err).ToNot(HaveOccurred())
	tmpFile.Close()

	Expect(bosh.Command("deployment", tmpFile.Name()).Wait(time.Second * 10)).To(Exit(0))

	tmpFile, err = os.Open(tmpFile.Name())
	Expect(err).ToNot(HaveOccurred())

	decoder := candiedyaml.NewDecoder(tmpFile)
	err = decoder.Decode(manifest)
	Expect(err).ToNot(HaveOccurred())
}
