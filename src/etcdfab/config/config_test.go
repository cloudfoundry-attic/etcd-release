package config_test

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/cloudfoundry-incubator/etcd-release/src/etcdfab/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func writeConfigurationFile(tmpDir, name string, configuration map[string]interface{}) string {
	file, err := ioutil.TempFile(tmpDir, name)
	Expect(err).NotTo(HaveOccurred())

	err = file.Close()
	Expect(err).NotTo(HaveOccurred())

	fileName := file.Name()

	configData, err := json.Marshal(configuration)
	Expect(err).NotTo(HaveOccurred())

	err = ioutil.WriteFile(fileName, configData, os.ModePerm)
	Expect(err).NotTo(HaveOccurred())

	return fileName
}

var _ = Describe("Config", func() {
	Describe("ConfigFromJSONs", func() {
		var (
			configFilePath     string
			linkConfigFilePath string
		)

		BeforeEach(func() {
			tmpDir, err := ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			configuration := map[string]interface{}{
				"node": map[string]interface{}{
					"name":        "some_name",
					"index":       3,
					"external_ip": "some-external-ip",
				},
				"etcd": map[string]interface{}{
					"heartbeat_interval_in_milliseconds": 10,
					"election_timeout_in_milliseconds":   20,
					"peer_require_ssl":                   false,
					"peer_ip":                            "some-peer-ip",
					"require_ssl":                        false,
					"client_ip":                          "some-client-ip",
					"advertise_urls_dns_suffix":          "some-dns-suffix",
				},
			}
			configFilePath = writeConfigurationFile(tmpDir, "config-file", configuration)

			linkConfiguration := map[string]interface{}{
				"machines": []string{
					"some-ip-1",
					"some-ip-2",
					"some-ip-3",
				},
				"heartbeat_interval_in_milliseconds": 10,
				"election_timeout_in_milliseconds":   33,
				"peer_require_ssl":                   false,
				"peer_ip":                            "some-peer-ip-from-link",
				"require_ssl":                        false,
				"client_ip":                          "some-client-ip-from-link",
				"advertise_urls_dns_suffix":          "some-dns-suffix-from-link",
			}
			linkConfigFilePath = writeConfigurationFile(tmpDir, "link-config-file", linkConfiguration)
		})

		It("returns a configuration populated with values from the specified files", func() {
			cfg, err := config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(cfg).To(Equal(config.Config{
				Node: config.Node{
					Name:       "some_name",
					Index:      3,
					ExternalIP: "some-external-ip",
				},
				Etcd: config.Etcd{
					EtcdPath:               "/var/vcap/packages/etcd/etcd",
					RunDir:                 "/var/vcap/sys/run/etcd",
					CertDir:                "/var/vcap/jobs/etcd/config/certs",
					DataDir:                "/var/vcap/store/etcd",
					HeartbeatInterval:      10,
					ElectionTimeout:        33,
					PeerRequireSSL:         false,
					PeerIP:                 "some-peer-ip-from-link",
					RequireSSL:             false,
					ClientIP:               "some-client-ip-from-link",
					AdvertiseURLsDNSSuffix: "some-dns-suffix-from-link",
					Machines:               []string{"some-ip-1", "some-ip-2", "some-ip-3"},
				},
			}))
		})

		Context("when there is no data in the link file", func() {
			BeforeEach(func() {
				err := ioutil.WriteFile(linkConfigFilePath, []byte("{}"), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns a configuration populated with values from the specified files", func() {
				cfg, err := config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
				Expect(err).NotTo(HaveOccurred())

				Expect(cfg).To(Equal(config.Config{
					Node: config.Node{
						Name:       "some_name",
						Index:      3,
						ExternalIP: "some-external-ip",
					},
					Etcd: config.Etcd{
						EtcdPath:               "/var/vcap/packages/etcd/etcd",
						RunDir:                 "/var/vcap/sys/run/etcd",
						CertDir:                "/var/vcap/jobs/etcd/config/certs",
						DataDir:                "/var/vcap/store/etcd",
						HeartbeatInterval:      10,
						ElectionTimeout:        20,
						PeerRequireSSL:         false,
						PeerIP:                 "some-peer-ip",
						RequireSSL:             false,
						ClientIP:               "some-client-ip",
						AdvertiseURLsDNSSuffix: "some-dns-suffix",
					},
				}))
			})
		})

		It("defaults values that are not specified in the JSON file", func() {
			err := ioutil.WriteFile(configFilePath, []byte("{}"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			err = ioutil.WriteFile(linkConfigFilePath, []byte("{}"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			cfg, err := config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
			Expect(err).NotTo(HaveOccurred())

			Expect(cfg.Etcd.EtcdPath).To(Equal("/var/vcap/packages/etcd/etcd"))
			Expect(cfg.Etcd.CertDir).To(Equal("/var/vcap/jobs/etcd/config/certs"))
			Expect(cfg.Etcd.RunDir).To(Equal("/var/vcap/sys/run/etcd"))
			Expect(cfg.Etcd.DataDir).To(Equal("/var/vcap/store/etcd"))
		})

		Context("failure cases", func() {
			Context("when it cannot read the config file", func() {
				It("returns the error to the caller and logs a helpful message", func() {
					_, err := config.ConfigFromJSONs("/path/to/missing/config", linkConfigFilePath)
					Expect(err).To(MatchError("error reading config file: open /path/to/missing/config: no such file or directory"))
				})
			})

			Context("when it cannot read the link config file", func() {
				It("returns the error to the caller and logs a helpful message", func() {
					_, err := config.ConfigFromJSONs(configFilePath, "/path/to/missing/config")
					Expect(err).To(MatchError("error reading link config file: open /path/to/missing/config: no such file or directory"))
				})
			})

			Context("when it cannot unmarshal the config file", func() {
				BeforeEach(func() {
					err := ioutil.WriteFile(configFilePath, []byte("%%%"), os.ModePerm)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns the error to the caller and logs a helpful message", func() {
					_, err := config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
					Expect(err).To(MatchError("invalid character '%' looking for beginning of value"))
				})
			})

			Context("when it cannot unmarshal the link config file", func() {
				BeforeEach(func() {
					err := ioutil.WriteFile(linkConfigFilePath, []byte("%%%"), os.ModePerm)
					Expect(err).NotTo(HaveOccurred())
				})

				It("returns the error to the caller and logs a helpful message", func() {
					_, err := config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
					Expect(err).To(MatchError("invalid character '%' looking for beginning of value"))
				})
			})
		})
	})

	Describe("NodeName", func() {
		var (
			cfg config.Config
		)

		BeforeEach(func() {
			tmpDir, err := ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			configuration := map[string]interface{}{
				"node": map[string]interface{}{
					"name":  "some_name",
					"index": 3,
				},
			}
			configFilePath := writeConfigurationFile(tmpDir, "config-file", configuration)

			linkConfigFilePath := writeConfigurationFile(tmpDir, "link-config-file", map[string]interface{}{})

			cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns the node name based on config", func() {
			Expect(cfg.NodeName()).To(Equal("some-name-3"))
		})
	})

	Describe("RequireSSL", func() {
		Context("when require_ssl is false", func() {
			var (
				cfg config.Config
			)

			BeforeEach(func() {
				tmpDir, err := ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())

				configuration := map[string]interface{}{
					"etcd": map[string]interface{}{
						"require_ssl": false,
					},
				}
				configFilePath := writeConfigurationFile(tmpDir, "config-file", configuration)

				linkConfigFilePath := writeConfigurationFile(tmpDir, "link-config-file", map[string]interface{}{})

				cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns false", func() {
				Expect(cfg.RequireSSL()).To(Equal(false))
			})
		})

		Context("when require_ssl is true", func() {
			var (
				cfg config.Config
			)

			BeforeEach(func() {
				tmpDir, err := ioutil.TempDir("", "")
				Expect(err).NotTo(HaveOccurred())

				configuration := map[string]interface{}{
					"etcd": map[string]interface{}{
						"require_ssl": true,
					},
				}
				configFilePath := writeConfigurationFile(tmpDir, "config-file", configuration)

				linkConfigFilePath := writeConfigurationFile(tmpDir, "link-config-file", map[string]interface{}{})

				cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns true", func() {
				Expect(cfg.RequireSSL()).To(Equal(true))
			})
		})
	})

	Describe("CertDir", func() {
		It("returns the CertDir", func() {
			cfg := config.Config{
				Etcd: config.Etcd{
					CertDir: "/var/vcap/jobs/etcd/config/certs",
				},
			}
			Expect(cfg.CertDir()).To(Equal("/var/vcap/jobs/etcd/config/certs"))
		})
	})

	Describe("AdvertisePeerURL", func() {
		var (
			cfg                config.Config
			configFilePath     string
			linkConfigFilePath string
		)

		BeforeEach(func() {
			tmpDir, err := ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			configuration := map[string]interface{}{
				"node": map[string]interface{}{
					"name":        "some_name",
					"index":       3,
					"external_ip": "some-external-ip",
				},
			}
			configFilePath = writeConfigurationFile(tmpDir, "config-file", configuration)

			linkConfigFilePath = writeConfigurationFile(tmpDir, "link-config-file", map[string]interface{}{})

			cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns the advertise peer url based on config", func() {
			Expect(cfg.AdvertisePeerURL()).To(Equal("http://some-external-ip:7001"))
		})

		Context("when PeerRequireSSL is true", func() {
			BeforeEach(func() {
				configuration := map[string]interface{}{
					"node": map[string]interface{}{
						"name":  "some_name",
						"index": 3,
					},
					"etcd": map[string]interface{}{
						"peer_require_ssl":          true,
						"advertise_urls_dns_suffix": "some-dns-suffix",
					},
				}
				configData, err := json.Marshal(configuration)
				Expect(err).NotTo(HaveOccurred())

				err = ioutil.WriteFile(configFilePath, configData, os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the advertise peer url based on config", func() {
				Expect(cfg.AdvertisePeerURL()).To(Equal("https://some-name-3.some-dns-suffix:7001"))
			})
		})
	})

	Describe("AdvertiseClientURL", func() {
		var (
			cfg                config.Config
			tmpDir             string
			configFilePath     string
			linkConfigFilePath string
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			configuration := map[string]interface{}{
				"node": map[string]interface{}{
					"name":        "some_name",
					"index":       3,
					"external_ip": "some-external-ip",
				},
			}
			configFilePath = writeConfigurationFile(tmpDir, "config-file", configuration)

			linkConfigFilePath = writeConfigurationFile(tmpDir, "link-config-file", map[string]interface{}{})

			cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns the advertise peer url based on config", func() {
			Expect(cfg.AdvertiseClientURL()).To(Equal("http://some-external-ip:4001"))
		})

		Context("when RequireSSL is true", func() {
			BeforeEach(func() {
				configuration := map[string]interface{}{
					"node": map[string]interface{}{
						"name":  "some_name",
						"index": 3,
					},
					"etcd": map[string]interface{}{
						"require_ssl":               true,
						"advertise_urls_dns_suffix": "some-dns-suffix",
					},
				}
				configFilePath = writeConfigurationFile(tmpDir, "config-file", configuration)

				var err error
				cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the advertise client url based on config", func() {
				Expect(cfg.AdvertiseClientURL()).To(Equal("https://some-name-3.some-dns-suffix:4001"))
			})
		})
	})

	Describe("ListenPeerURL", func() {
		var (
			cfg                config.Config
			tmpDir             string
			configFilePath     string
			linkConfigFilePath string
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			configuration := map[string]interface{}{
				"node": map[string]interface{}{
					"name":  "some_name",
					"index": 3,
				},
				"etcd": map[string]interface{}{
					"peer_ip": "some-peer-ip",
				},
			}
			configFilePath = writeConfigurationFile(tmpDir, "config-file", configuration)

			linkConfigFilePath = writeConfigurationFile(tmpDir, "link-config-file", map[string]interface{}{})

			cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns the listen peer url based on config", func() {
			Expect(cfg.ListenPeerURL()).To(Equal("http://some-peer-ip:7001"))
		})

		Context("when PeerRequireSSL is true", func() {
			BeforeEach(func() {
				configuration := map[string]interface{}{
					"node": map[string]interface{}{
						"name":  "some_name",
						"index": 3,
					},
					"etcd": map[string]interface{}{
						"peer_require_ssl": true,
						"peer_ip":          "some-peer-ip",
					},
				}
				configFilePath = writeConfigurationFile(tmpDir, "config-file", configuration)

				var err error
				cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the listen peer url based on config", func() {
				Expect(cfg.ListenPeerURL()).To(Equal("https://some-peer-ip:7001"))
			})
		})
	})

	Describe("ListenClientURL", func() {
		var (
			cfg                config.Config
			tmpDir             string
			configFilePath     string
			linkConfigFilePath string
		)

		BeforeEach(func() {
			var err error
			tmpDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			configuration := map[string]interface{}{
				"node": map[string]interface{}{
					"name":  "some_name",
					"index": 3,
				},
				"etcd": map[string]interface{}{
					"client_ip": "some-client-ip",
				},
			}
			configFilePath = writeConfigurationFile(tmpDir, "config-file", configuration)

			linkConfigFilePath = writeConfigurationFile(tmpDir, "link-config-file", map[string]interface{}{})

			cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns the listen peer url based on config", func() {
			Expect(cfg.ListenClientURL()).To(Equal("http://some-client-ip:4001"))
		})

		Context("when RequireSSL is true", func() {
			BeforeEach(func() {
				configuration := map[string]interface{}{
					"node": map[string]interface{}{
						"name":  "some_name",
						"index": 3,
					},
					"etcd": map[string]interface{}{
						"require_ssl": true,
						"client_ip":   "some-client-ip",
					},
				}
				configFilePath = writeConfigurationFile(tmpDir, "config-file", configuration)

				var err error
				cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the listen peer url based on config", func() {
				Expect(cfg.ListenClientURL()).To(Equal("https://some-client-ip:4001"))
			})
		})
	})

	Describe("EtcdClientEndpoints", func() {
		var (
			cfg                config.Config
			configFilePath     string
			linkConfigFilePath string
		)

		BeforeEach(func() {
			tmpDir, err := ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			configuration := map[string]interface{}{
				"etcd": map[string]interface{}{
					"require_ssl":      false,
					"peer_require_ssl": false,
					"machines":         []string{"some-ip-1", "some-ip-2"},
				},
			}
			configFilePath = writeConfigurationFile(tmpDir, "config-file", configuration)

			linkConfigFilePath = writeConfigurationFile(tmpDir, "link-config-file", map[string]interface{}{})

			cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns the etcd client endpoints based on config", func() {
			Expect(cfg.EtcdClientEndpoints()).To(Equal([]string{"http://some-ip-1:4001", "http://some-ip-2:4001"}))
		})

		Context("when RequireSSL or PeerRequireSSL is true", func() {
			BeforeEach(func() {
				configuration := map[string]interface{}{
					"etcd": map[string]interface{}{
						"require_ssl":               true,
						"peer_require_ssl":          true,
						"advertise_urls_dns_suffix": "some-dns-suffix",
					},
				}
				configData, err := json.Marshal(configuration)
				Expect(err).NotTo(HaveOccurred())

				err = ioutil.WriteFile(configFilePath, configData, os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				cfg, err = config.ConfigFromJSONs(configFilePath, linkConfigFilePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the etcd client endpoints with the correct protocol based on config", func() {
				Expect(cfg.EtcdClientEndpoints()).To(Equal([]string{"https://some-dns-suffix:4001"}))
			})
		})
	})
})
