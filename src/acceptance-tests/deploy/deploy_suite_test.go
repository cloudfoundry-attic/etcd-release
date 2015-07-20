package deploy_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var (
	gemfilePath string
	suitePath   string
)

func TestDeploy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Deploy Suite")
}

var _ = BeforeSuite(func() {
	gopath := os.Getenv("GOPATH")
	suitePath = strings.Split(gopath, ":")[0]

	wd, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	gemfilePath = filepath.Join(wd, "..", "Gemfile")

	cmd := exec.Command("bundle")
	env := os.Environ()
	cmd.Env = append(env, fmt.Sprintf("BUNDLE_GEMFILE=%s", gemfilePath))
	// fmt.Println(cmd.Env)

	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
})
