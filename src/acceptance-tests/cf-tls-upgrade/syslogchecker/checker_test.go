package syslogchecker_test

import (
	"acceptance-tests/cf-tls-upgrade/syslogchecker"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeGuidGenerator struct{}

func (fakeGuidGenerator) Generate() string {
	return "some-guid"
}

var _ = FDescribe("Checker", func() {
	var (
		oldPath           string
		logspinnerServer  *httptest.Server
		mutex             sync.Mutex
		messages          []string
		guidGenerator     fakeGuidGenerator
		stopLogging       bool
		logSpinnerAppName string
		checker           syslogchecker.Checker
	)

	var addMessage = func(message string) {
		mutex.Lock()
		defer mutex.Unlock()

		messages = append(messages, message)
	}

	var getMessages = func() []string {
		mutex.Lock()
		defer mutex.Unlock()

		copy := []string{}
		for _, m := range messages {
			copy = append(copy, m)
		}

		return copy
	}

	AfterEach(func() {
		os.Setenv("PATH", oldPath)
		err := ioutil.WriteFile(pathToRecentLogs, []byte(""), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {
		oldPath = os.Getenv("PATH")
		os.Setenv("PATH", fmt.Sprintf("%s:%s", filepath.Dir(pathToCF), oldPath))

		guidGenerator = fakeGuidGenerator{}
		messages = []string{}
		stopLogging = false

		logspinnerServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasPrefix(req.URL.Path, "/log") && req.Method == "GET" {
				w.WriteHeader(http.StatusOK)
				parts := strings.Split(req.URL.Path, "/")
				message := parts[2]
				addMessage(message)

				if !stopLogging {
					err := ioutil.WriteFile(pathToRecentLogs, []byte(message), os.ModePerm)
					Expect(err).NotTo(HaveOccurred())
				}

				return
			}
			w.WriteHeader(http.StatusTeapot)
		}))

		err := ioutil.WriteFile(pathToCFOutput, []byte("[]"), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		logSpinnerAppName = fmt.Sprintf("my-app-%v", rand.Int())
		checker = syslogchecker.New("syslog-app", guidGenerator, 1*time.Second)
	})

	Describe("Check", func() {
		It("returns any errors that occurred during checker", func() {
			watcher := checker.Start(logSpinnerAppName, "not-a-real-app")

			Expect(<-watcher).To(BeTrue())

			err := checker.Stop()
			Expect(err).NotTo(HaveOccurred())

			Expect(checker.Check()).Should(HaveKey(
				`Get not-a-real-app/log/some-guid: unsupported protocol scheme ""`,
			))
		})
	})

	Describe("Start", func() {
		It("streams logs from an app to the syslog listener", func() {
			sysLogAppName := "syslog-app-some-guid"

			watcher := checker.Start(logSpinnerAppName, logspinnerServer.URL)

			Expect(<-watcher).To(BeTrue())

			err := checker.Stop()
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]byte, error) {
				return ioutil.ReadFile(pathToCFOutput)
			}).Should(MatchJSON(fmt.Sprintf(`[
				["push",%[3]q,"-f","assets/syslog-drainer/manifest.yml", "--no-start"],
				["enable-diego", %[3]q],
				["start", %[3]q],
				["logs", %[3]q, "--recent"],
				["cups", "%[3]s-service", "-l", "syslog://127.0.0.1:%[2]s"],
				["bind-service", %[1]q, "%[3]s-service"],
				["restage", %[1]q],
				["logs", %[3]q, "--recent"],
				["unbind-service", %[1]q, "%[3]s-service"],
				["delete-service", "%[3]s-service", "-f"],
				["delete", %[3]q, "-f", "-r"]
			]`, logSpinnerAppName, syslogListenerPort, sysLogAppName)))
		})

		It("hits the logspinner app until stop is called", func() {
			watcher := checker.Start(logSpinnerAppName, logspinnerServer.URL)

			Expect(<-watcher).To(BeTrue())
			Expect(<-watcher).To(BeTrue())

			err := checker.Stop()
			Expect(err).NotTo(HaveOccurred())

			Expect(getMessages()).To(Equal([]string{
				"some-guid",
				"some-guid",
			}))
		})

		Context("failure cases", func() {
			It("records an error when the syslog listener fails to validate it got the guid", func() {
				stopLogging = true
				watcher := checker.Start(logSpinnerAppName, logspinnerServer.URL)

				Expect(<-watcher).To(BeTrue())

				err := checker.Stop()
				Expect(err).NotTo(HaveOccurred())

				Expect(checker.Check()).To(HaveKey("could not validate the guid on syslog"))
			})

			It("records the error in case the cf command fails", func() {
				err := ioutil.WriteFile(pathToCFOutput, []byte(""), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
				watcher := checker.Start(logSpinnerAppName, logspinnerServer.URL)

				Expect(<-watcher).To(BeTrue())

				err = checker.Stop()
				Expect(err).NotTo(HaveOccurred())

				Expect(checker.Check()).To(HaveKey("syslog drainer application push failed"))
			})
		})
	})

	Describe("Stop", func() {
		It("stops the checker and no longer hits the log spinner", func() {
			_ = checker.Start(logSpinnerAppName, logspinnerServer.URL)

			err := checker.Stop()
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() ([]byte, error) {
				return ioutil.ReadFile(pathToCFOutput)
			}).Should(Equal([]byte(`[]`)))
		})
	})
})
