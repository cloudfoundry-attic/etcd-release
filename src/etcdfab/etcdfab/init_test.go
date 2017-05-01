package main_test

import (
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes/etcd/backend"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func TestEtcdFab(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "etcdfab")
}

var (
	etcdBackendServer *backend.EtcdBackendServer

	pathToFakeEtcd string
	pathToEtcdFab  string
)

var _ = BeforeSuite(func() {
	etcdBackendServer = backend.NewEtcdBackendServer()

	var err error
	pathToFakeEtcd, err = gexec.Build("github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/fakes/etcd",
		"--ldflags", fmt.Sprintf("-X main.backendURL=%s", etcdBackendServer.ServerURL()))
	Expect(err).NotTo(HaveOccurred())

	pathToEtcdFab, err = gexec.Build("github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/etcdfab")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

func etcdFab(args []string, exitCode int) *gexec.Session {
	command := exec.Command(pathToEtcdFab, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session, 10*time.Second).Should(gexec.Exit(exitCode))

	return session
}
