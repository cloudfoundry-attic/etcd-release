package main_test

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("provides an http interface to the etcd cluster", func() {
	var (
		session *gexec.Session
		port    string
		handler http.HandlerFunc
	)

	BeforeEach(func() {
		var err error
		port, err = openPort()
		Expect(err).NotTo(HaveOccurred())
		handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.Method {
			case "GET":
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
						"action": "get",
						"node": {
							"createdIndex": 2,
							"key": "/some-key",
							"modifiedIndex": 2,
							"value": "some-value"
						}
					}`))
				return
			case "PUT":
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{
						"action": "set",
						"node": {
							"createdIndex": 2,
							"key": "/some-key",
							"modifiedIndex": 2,
							"value": "some-value"
						}
					}`))
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
		})
	})

	AfterEach(func() {
		session.Terminate().Wait()
	})

	Context("main", func() {
		It("allows multiple etcd-service urls", func() {
			etcdServer1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
						"action": "get",
						"node": {
							"createdIndex": 2,
							"key": "/some-key",
							"modifiedIndex": 2,
							"value": "server1"
						}
					}`))
			}))

			etcdServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
						"action": "get",
						"node": {
							"createdIndex": 2,
							"key": "/some-key",
							"modifiedIndex": 2,
							"value": "server2"
						}
					}`))
			}))

			command := exec.Command(pathToConsumer, "--port", port,
				"--etcd-service", etcdServer1.URL,
				"--etcd-service", etcdServer2.URL,
			)

			var err error
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			waitForServerToStart(port)

			Eventually(func() (string, error) {
				_, body, err := makeRequest("GET", fmt.Sprintf("http://localhost:%s/kv/some-key", port), "")
				return body, err
			}).Should(Equal("server1"))

			Eventually(func() (string, error) {
				_, body, err := makeRequest("GET", fmt.Sprintf("http://localhost:%s/kv/some-key", port), "")
				return body, err
			}).Should(Equal("server2"))
		})
	})

	Context("kv", func() {
		Context("etcd ssl mode", func() {
			BeforeEach(func() {
				var err error
				etcdServer := httptest.NewUnstartedServer(handler)
				etcdServer.TLS = &tls.Config{}
				etcdServer.TLS.Certificates = make([]tls.Certificate, 1)
				etcdServer.TLS.Certificates[0], err = tls.LoadX509KeyPair("fixtures/server.crt", "fixtures/server.key")
				Expect(err).NotTo(HaveOccurred())

				etcdServer.StartTLS()

				command := exec.Command(pathToConsumer,
					"--port", port,
					"--etcd-service", etcdServer.URL,
					"--ca-cert-file", "fixtures/ca.crt",
					"--client-ssl-cert-file", "fixtures/client.crt",
					"--client-ssl-key-file", "fixtures/client.key",
				)

				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				waitForServerToStart(port)
			})

			Context("GET", func() {
				It("returns a value with the given key", func() {
					status, body, err := makeRequest("GET", fmt.Sprintf("http://localhost:%s/kv/some-key", port), "")
					Expect(err).NotTo(HaveOccurred())
					Expect(status).To(Equal(http.StatusOK))
					Expect(body).To(Equal("some-value"))
				})
			})

			Context("PUT", func() {
				It("sets a value with the given key", func() {
					status, _, err := makeRequest("PUT", fmt.Sprintf("http://localhost:%s/kv/some-key", port), "some-value")
					Expect(err).NotTo(HaveOccurred())
					Expect(status).To(Equal(http.StatusCreated))
				})
			})
		})

		Context("etcd non-ssl mode", func() {
			BeforeEach(func() {
				etcdServer := httptest.NewServer(handler)

				command := exec.Command(pathToConsumer, "--port", port, "--etcd-service", etcdServer.URL)

				var err error
				session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				waitForServerToStart(port)
			})

			Context("GET", func() {
				It("returns a value with the given key", func() {
					status, body, err := makeRequest("GET", fmt.Sprintf("http://localhost:%s/kv/some-key", port), "")
					Expect(err).NotTo(HaveOccurred())
					Expect(status).To(Equal(http.StatusOK))
					Expect(body).To(Equal("some-value"))
				})
			})

			Context("PUT", func() {
				It("sets a value with the given key", func() {
					status, _, err := makeRequest("PUT", fmt.Sprintf("http://localhost:%s/kv/some-key", port), "some-value")
					Expect(err).NotTo(HaveOccurred())
					Expect(status).To(Equal(http.StatusCreated))
				})
			})
		})
	})
})

func makeRequest(method string, url string, body string) (int, string, error) {
	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return 0, "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, "", err
	}

	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, "", err
	}

	return response.StatusCode, string(responseBody), nil
}

func openPort() (string, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", err
	}

	defer l.Close()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return "", err
	}

	return port, nil
}

func waitForServerToStart(port string) {
	timer := time.After(0 * time.Second)
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			panic("Failed to boot!")
		case <-timer:
			_, err := http.Get("http://localhost:" + port + "/kv/banana")
			if err == nil {
				return
			}

			timer = time.After(2 * time.Second)
		}
	}
}
