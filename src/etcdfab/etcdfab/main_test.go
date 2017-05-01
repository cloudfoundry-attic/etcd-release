package main_test

import (
	"fmt"
	"io/ioutil"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EtcdFab", func() {
	var pathToEtcdPid string

	BeforeEach(func() {
		tmpDir, err := ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		pathToEtcdPid = fmt.Sprintf("%s/etcd-pid", tmpDir)
	})

	It("shells out to etcd with provided flags", func() {
		_ = etcdFab([]string{
			pathToFakeEtcd,
			pathToEtcdPid,
			"--name", "some-name",
			"--data-dir", "some-data-dir",
			"--heartbeat-interval", "some-heartbeat-interval",
			"--election-timeout", "some-election-timeout",
			"--listen-peer-urls", "some-listen-peer-urls",
			"--listen-client-urls", "some-listen-client-urls",
			"--initial-advertise-peer-urls", "some-initial-advertise-peer-urls",
			"--advertise-client-urls", "some-advertise-client-urls",
			"--initial-cluster", "some-initial-cluster",
			"--initial-cluster-state", "some-initial-cluster-state",
		}, 0)

		Expect(etcdBackendServer.GetCallCount()).To(Equal(1))
		Expect(etcdBackendServer.GetArgs()).To(Equal([]string{
			"--name", "some-name",
			"--data-dir", "some-data-dir",
			"--heartbeat-interval", "some-heartbeat-interval",
			"--election-timeout", "some-election-timeout",
			"--listen-peer-urls", "some-listen-peer-urls",
			"--listen-client-urls", "some-listen-client-urls",
			"--initial-advertise-peer-urls", "some-initial-advertise-peer-urls",
			"--advertise-client-urls", "some-advertise-client-urls",
			"--initial-cluster", "some-initial-cluster",
			"--initial-cluster-state", "some-initial-cluster-state",
		}))
	})

	It("writes etcd stdout/stderr", func() {
		session := etcdFab([]string{
			pathToFakeEtcd,
			pathToEtcdPid,
			"--name", "some-name",
			"--data-dir", "some-data-dir",
			"--heartbeat-interval", "some-heartbeat-interval",
			"--election-timeout", "some-election-timeout",
			"--listen-peer-urls", "some-listen-peer-urls",
			"--listen-client-urls", "some-listen-client-urls",
			"--initial-advertise-peer-urls", "some-initial-advertise-peer-urls",
			"--advertise-client-urls", "some-advertise-client-urls",
			"--initial-cluster", "some-initial-cluster",
			"--initial-cluster-state", "some-initial-cluster-state",
		}, 0)

		Expect(session.Out.Contents()).To(ContainSubstring("starting fake etcd"))
		Expect(session.Out.Contents()).To(ContainSubstring("stopping fake etcd"))
		Expect(session.Err.Contents()).To(ContainSubstring("fake error in stderr"))
	})

	It("writes the pid of etcd to the file provided", func() {
		_ = etcdFab([]string{
			pathToFakeEtcd,
			pathToEtcdPid,
			"--name", "some-name",
			"--data-dir", "some-data-dir",
			"--heartbeat-interval", "some-heartbeat-interval",
			"--election-timeout", "some-election-timeout",
			"--listen-peer-urls", "some-listen-peer-urls",
			"--listen-client-urls", "some-listen-client-urls",
			"--initial-advertise-peer-urls", "some-initial-advertise-peer-urls",
			"--advertise-client-urls", "some-advertise-client-urls",
			"--initial-cluster", "some-initial-cluster",
			"--initial-cluster-state", "some-initial-cluster-state",
		}, 0)

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
				session := etcdFab([]string{
					"bogus",
					pathToEtcdPid,
				}, 1)

				Expect(session.Out.Contents()).To(MatchRegexp(`{"timestamp":".*","source":"etcdfab","message":"etcdfab\.main","log_level":2,"data":{"error":"exec: \\"bogus\\": executable file not found in \$PATH"}}`))
			})
		})
	})
})
