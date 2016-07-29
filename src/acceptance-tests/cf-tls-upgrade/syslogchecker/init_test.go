package syslogchecker_test

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	pathToCF           string
	pathToCFOutput     string
	pathToRecentLogs   string
	syslogListenerPort string
)

var _ = BeforeSuite(func() {
	var err error
	tmpFile, err := ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())

	pathToCFOutput = tmpFile.Name()

	tmpFile, err = ioutil.TempFile("", "")
	Expect(err).NotTo(HaveOccurred())
	pathToRecentLogs = tmpFile.Name()

	syslogListenerPort = fmt.Sprintf("%d", rand.Int())
	args := []string{
		"-ldflags",
		fmt.Sprintf(`-X "main.RecentLogsPath=%s" -X "main.OutputPath=%s" -X "main.SyslogListenerPort=%s"`, pathToRecentLogs, pathToCFOutput, syslogListenerPort),
	}

	pathToCF, err = gexec.Build("acceptance-tests/cf-tls-upgrade/syslogchecker/cf", args...)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func TestSyslogchecker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "syslogchecker")
}
