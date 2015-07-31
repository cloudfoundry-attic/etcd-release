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
	Properties Properties `yaml:"properties"`
}

type Properties struct {
	Etcd Etcd `yaml:"etcd"`
}

type Etcd struct {
	Machines []string `yaml:"machines"`
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
