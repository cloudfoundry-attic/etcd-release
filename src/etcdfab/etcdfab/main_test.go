package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("EtcdFab", func() {
	var (
		pathToEtcdPid string
		configFile    *os.File
	)

	BeforeEach(func() {
		tmpDir, err := ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		pathToEtcdPid = fmt.Sprintf("%s/etcd-pid", tmpDir)

		configFile, err = ioutil.TempFile(tmpDir, "config-file")
		Expect(err).NotTo(HaveOccurred())

		err = configFile.Close()
		Expect(err).NotTo(HaveOccurred())

		writeConfigurationFile(configFile.Name(), map[string]interface{}{
			"node": map[string]interface{}{
				"name":  "some_name",
				"index": 3,
			},
			"etcd": map[string]interface{}{
				"heartbeat_interval_in_milliseconds": 10,
				"election_timeout_in_milliseconds":   20,
			},
		})
	})

	AfterEach(func() {
		Expect(os.Remove(configFile.Name())).NotTo(HaveOccurred())
	})

	It("shells out to etcd with provided flags", func() {
		command := exec.Command(pathToEtcdFab,
			pathToFakeEtcd,
			pathToEtcdPid,
			configFile.Name(),
			"--listen-peer-urls", "some-listen-peer-urls",
			"--listen-client-urls", "some-listen-client-urls",
			"--initial-advertise-peer-urls", "some-initial-advertise-peer-urls",
			"--advertise-client-urls", "some-advertise-client-urls",
			"--initial-cluster", "some-initial-cluster",
			"--initial-cluster-state", "some-initial-cluster-state",
		)
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, 10*time.Second).Should(gexec.Exit(0))

		Expect(etcdBackendServer.GetCallCount()).To(Equal(1))
		Expect(etcdBackendServer.GetArgs()).To(Equal([]string{
			"--listen-peer-urls", "some-listen-peer-urls",
			"--listen-client-urls", "some-listen-client-urls",
			"--initial-advertise-peer-urls", "some-initial-advertise-peer-urls",
			"--advertise-client-urls", "some-advertise-client-urls",
			"--initial-cluster", "some-initial-cluster",
			"--initial-cluster-state", "some-initial-cluster-state",
			"--name", "some-name-3",
			"--data-dir", "/var/vcap/store/etcd",
			"--heartbeat-interval", "10",
			"--election-timeout", "20",
		}))
	})

	It("writes etcd stdout/stderr", func() {
		command := exec.Command(pathToEtcdFab,
			pathToFakeEtcd,
			pathToEtcdPid,
			configFile.Name(),
			"--listen-peer-urls", "some-listen-peer-urls",
			"--listen-client-urls", "some-listen-client-urls",
			"--initial-advertise-peer-urls", "some-initial-advertise-peer-urls",
			"--advertise-client-urls", "some-advertise-client-urls",
			"--initial-cluster", "some-initial-cluster",
			"--initial-cluster-state", "some-initial-cluster-state",
		)
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, 10*time.Second).Should(gexec.Exit(0))

		Expect(session.Out.Contents()).To(ContainSubstring("starting fake etcd"))
		Expect(session.Out.Contents()).To(ContainSubstring("stopping fake etcd"))
		Expect(session.Err.Contents()).To(ContainSubstring("fake error in stderr"))
	})

	It("writes the pid of etcd to the file provided", func() {
		command := exec.Command(pathToEtcdFab,
			pathToFakeEtcd,
			pathToEtcdPid,
			configFile.Name(),
			"--listen-peer-urls", "some-listen-peer-urls",
			"--listen-client-urls", "some-listen-client-urls",
			"--initial-advertise-peer-urls", "some-initial-advertise-peer-urls",
			"--advertise-client-urls", "some-advertise-client-urls",
			"--initial-cluster", "some-initial-cluster",
			"--initial-cluster-state", "some-initial-cluster-state",
		)
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, 10*time.Second).Should(gexec.Exit(0))

		Expect(pathToEtcdPid).To(BeARegularFile())

		etcdPid, err := ioutil.ReadFile(pathToEtcdPid)
		Expect(err).NotTo(HaveOccurred())

		Expect(strconv.Atoi(string(etcdPid))).To(SatisfyAll(
			BeNumerically(">", 0),
			BeNumerically("<", 4194304),
		))
	})

	Context("failure cases", func() {
		Context("when the etcd process fails", func() {
			BeforeEach(func() {
				etcdBackendServer.EnableFastFail()
			})

			AfterEach(func() {
				etcdBackendServer.DisableFastFail()
			})

			It("exits 1 and prints an error", func() {
				command := exec.Command(pathToEtcdFab,
					"bogus",
					pathToEtcdPid,
					configFile.Name(),
				)
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, 10*time.Second).Should(gexec.Exit(1))

				Expect(session.Out.Contents()).To(MatchRegexp(`{"timestamp":".*","source":"etcdfab","message":"etcdfab\.main","log_level":2,"data":{"error":"exec: \\"bogus\\": executable file not found in \$PATH"}}`))
			})
		})
	})
})
