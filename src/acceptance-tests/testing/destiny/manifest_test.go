package destiny_test

import (
	"acceptance-tests/testing/destiny"
	"io/ioutil"

	. "acceptance-tests/testing/matchers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifest", func() {
	Describe("ToYAML", func() {
		It("returns a YAML representation of the etcd manifest", func() {
			etcdManifest, err := ioutil.ReadFile("fixtures/etcd_manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			manifest := destiny.NewEtcd(destiny.Config{
				DirectorUUID: "some-director-uuid",
				Name:         "etcd",
				IAAS:         destiny.Warden,
			})

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml).To(MatchYAML(etcdManifest))
		})

		It("returns a YAML representation of the turbulence manifest", func() {
			turbulenceManifest, err := ioutil.ReadFile("fixtures/turbulence_manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			manifest := destiny.NewTurbulence(destiny.Config{
				DirectorUUID: "some-director-uuid",
				Name:         "turbulence",
				IAAS:         destiny.Warden,
				BOSH: destiny.ConfigBOSH{
					Target:   "some-bosh-target",
					Username: "some-bosh-username",
					Password: "some-bosh-password",
				},
			})

			yaml, err := manifest.ToYAML()
			Expect(err).NotTo(HaveOccurred())
			Expect(yaml).To(MatchYAML(turbulenceManifest))
		})
	})

	Describe("FromYAML", func() {
		It("returns a Manifest matching the given YAML", func() {
			etcdManifest, err := ioutil.ReadFile("fixtures/etcd_manifest.yml")
			Expect(err).NotTo(HaveOccurred())

			manifest, err := destiny.FromYAML(etcdManifest)
			Expect(err).NotTo(HaveOccurred())

			Expect(manifest).To(Equal(destiny.Manifest{
				DirectorUUID: "some-director-uuid",
				Name:         "etcd",
				Releases: []destiny.Release{{
					Name:    "etcd",
					Version: "latest",
				}},
				Compilation: destiny.Compilation{
					Network:             "etcd1",
					ReuseCompilationVMs: true,
					Workers:             3,
				},
				Update: destiny.Update{
					Canaries:        1,
					CanaryWatchTime: "1000-180000",
					MaxInFlight:     1,
					Serial:          true,
					UpdateWatchTime: "1000-180000",
				},
				ResourcePools: []destiny.ResourcePool{
					{
						Name:    "etcd_z1",
						Network: "etcd1",
						Stemcell: destiny.ResourcePoolStemcell{
							Name:    "bosh-warden-boshlite-ubuntu-trusty-go_agent",
							Version: "latest",
						},
					},
				},
				Jobs: []destiny.Job{
					{
						Name:      "etcd_z1",
						Instances: 1,
						Networks: []destiny.JobNetwork{{
							Name:      "etcd1",
							StaticIPs: []string{"10.244.4.4"},
						}},
						PersistentDisk: 1024,
						ResourcePool:   "etcd_z1",
						Templates: []destiny.JobTemplate{{
							Name:    "etcd",
							Release: "etcd",
						}},
					},
				},
				Networks: []destiny.Network{
					{
						Name: "etcd1",
						Subnets: []destiny.NetworkSubnet{
							{
								CloudProperties: destiny.NetworkSubnetCloudProperties{Name: "random"},
								Gateway:         "10.244.4.1",
								Range:           "10.244.4.0/24",
								Reserved: []string{
									"10.244.4.2-10.244.4.3",
									"10.244.4.12-10.244.4.254",
								},
								Static: []string{
									"10.244.4.4",
									"10.244.4.5",
									"10.244.4.6",
									"10.244.4.7",
									"10.244.4.8",
								},
							},
						},
						Type: "manual",
					},
				},
				Properties: destiny.Properties{
					Etcd: &destiny.PropertiesEtcd{
						Machines: []string{
							"10.244.4.4",
						},
						PeerRequireSSL:                  false,
						RequireSSL:                      false,
						HeartbeatIntervalInMilliseconds: 50,
					},
				},
			}))
		})

		Context("failure cases", func() {
			It("should error on malformed YAML", func() {
				_, err := destiny.FromYAML([]byte("%%%%%%%%%%"))
				Expect(err).To(MatchError(ContainSubstring("yaml: ")))
			})
		})
	})
})
