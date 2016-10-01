package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IPTablesAgent", func() {
	var (
		sinkServer      *httptest.Server
		agentPort       string
		lastSinkRequest string
	)

	BeforeEach(func() {
		sinkServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqContents, err := ioutil.ReadAll(r.Body)
			Expect(err).NotTo(HaveOccurred())

			lastSinkRequest = string(reqContents)
		}))

		buildFakeIPTables(sinkServer.URL, false)

		var err error
		agentPort, err = openPort()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when /drop?addr={ip}&port={port} is called", func() {
		It("applies iptables drop output request", func() {
			command := exec.Command(pathToIPTablesAgent, "--port", agentPort)

			_, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			waitForServerToStart(agentPort)

			req, err := http.NewRequest("PUT", fmt.Sprintf("http://localhost:%s/drop?addr=some-ip-addr&port=9898", agentPort), strings.NewReader(""))
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			respContents, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(respContents).To(BeEmpty())

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			Expect(lastSinkRequest).To(Equal("iptables -A OUTPUT -p tcp -d some-ip-addr --dport 9898 -j DROP"))
		})
	})

	Context("failure cases", func() {
		BeforeEach(func() {
			buildFakeIPTables(sinkServer.URL, true)
		})

		It("responds with internal server error and stdout when iptables fails", func() {
			command := exec.Command(pathToIPTablesAgent, "--port", agentPort)

			_, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			waitForServerToStart(agentPort)

			req, err := http.NewRequest("PUT", fmt.Sprintf("http://localhost:%s/drop?addr=some-ip-addr&port=9898", agentPort), strings.NewReader(""))
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

			respContents, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(respContents)).To(Equal("error: exit status 1\niptables output: fast failing..."))
		})
	})
})

func buildFakeIPTables(sinkServerURL string, fastFail bool) {
	args := []string{
		"-ldflags",
		fmt.Sprintf("-X main.SinkURL=%s", sinkServerURL),
	}

	if fastFail {
		args = []string{
			"-ldflags",
			fmt.Sprintf("-X main.SinkURL=%s -X main.Fail=true", sinkServerURL),
		}
	}

	pathToFakeIPTables, err := gexec.Build("acceptance-tests/testing/iptables_agent/fakeiptables", args...)
	Expect(err).NotTo(HaveOccurred())

	pathToIPTables := filepath.Join(filepath.Dir(pathToFakeIPTables), "iptables")
	err = os.Rename(pathToFakeIPTables, pathToIPTables)
	Expect(err).NotTo(HaveOccurred())

	os.Setenv("PATH", strings.Join([]string{filepath.Dir(pathToIPTables), os.Getenv("PATH")}, ":"))
}
