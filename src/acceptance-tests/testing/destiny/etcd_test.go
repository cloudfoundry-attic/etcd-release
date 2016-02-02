package destiny_test

import (
	"acceptance-tests/testing/destiny"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Etcd Manifest", func() {
	Describe("NewEtcd", func() {
		It("generates a valid Etcd BOSH-Lite manifest", func() {
			manifest := destiny.NewEtcd(destiny.Config{
				DirectorUUID: "some-director-uuid",
				Name:         "etcd-some-random-guid",
				IAAS:         destiny.Warden,
			})

			Expect(manifest).To(Equal(destiny.Manifest{
				DirectorUUID: "some-director-uuid",
				Name:         "etcd-some-random-guid",
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

		It("generates a valid Etcd AWS manifest", func() {
			manifest := destiny.NewEtcd(destiny.Config{
				DirectorUUID: "some-director-uuid",
				Name:         "etcd-some-random-guid",
				IAAS:         destiny.AWS,
				AWS: destiny.ConfigAWS{
					Subnet: "subnet-1234",
				},
			})

			Expect(manifest).To(Equal(destiny.Manifest{
				DirectorUUID: "some-director-uuid",
				Name:         "etcd-some-random-guid",
				Releases: []destiny.Release{{
					Name:    "etcd",
					Version: "latest",
				}},
				Compilation: destiny.Compilation{
					Network:             "etcd1",
					ReuseCompilationVMs: true,
					Workers:             3,
					CloudProperties: destiny.CompilationCloudProperties{
						InstanceType:     "m3.medium",
						AvailabilityZone: "us-east-1a",
						EphemeralDisk: &destiny.CompilationCloudPropertiesEphemeralDisk{
							Size: 1024,
							Type: "gp2",
						},
					},
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
							Name:    "bosh-aws-xen-hvm-ubuntu-trusty-go_agent",
							Version: "latest",
						},
						CloudProperties: destiny.ResourcePoolCloudProperties{
							InstanceType:     "m3.medium",
							AvailabilityZone: "us-east-1a",
							EphemeralDisk: &destiny.ResourcePoolCloudPropertiesEphemeralDisk{
								Size: 1024,
								Type: "gp2",
							},
						},
					},
				},
				Jobs: []destiny.Job{
					{
						Name:      "etcd_z1",
						Instances: 1,
						Networks: []destiny.JobNetwork{{
							Name:      "etcd1",
							StaticIPs: []string{"10.0.16.4"},
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
								CloudProperties: destiny.NetworkSubnetCloudProperties{Subnet: "subnet-1234"},
								Gateway:         "10.0.16.1",
								Range:           "10.0.16.0/24",
								Reserved: []string{
									"10.0.16.2-10.0.16.3",
									"10.0.16.12-10.0.16.254",
								},
								Static: []string{
									"10.0.16.4",
									"10.0.16.5",
									"10.0.16.6",
									"10.0.16.7",
									"10.0.16.8",
								},
							},
						},
						Type: "manual",
					},
				},
				Properties: destiny.Properties{
					Etcd: &destiny.PropertiesEtcd{
						Machines: []string{
							"10.0.16.4",
						},
						PeerRequireSSL:                  false,
						RequireSSL:                      false,
						HeartbeatIntervalInMilliseconds: 50,
					},
				},
			}))
		})
	})

	Describe("EtcdMembers", func() {
		Context("when there is a single job with a single instance", func() {
			It("returns a list of members in the cluster", func() {
				manifest := destiny.Manifest{
					Jobs: []destiny.Job{
						{
							Instances: 1,
							Networks: []destiny.JobNetwork{{
								StaticIPs: []string{"10.244.4.2"},
							}},
						},
					},
				}

				members := manifest.EtcdMembers()
				Expect(members).To(Equal([]destiny.EtcdMember{{
					Address: "10.244.4.2",
				}}))
			})
		})

		Context("when there are multiple jobs with multiple instances", func() {
			It("returns a list of members in the cluster", func() {
				manifest := destiny.Manifest{
					Jobs: []destiny.Job{
						{
							Instances: 0,
						},
						{
							Instances: 1,
							Networks: []destiny.JobNetwork{{
								StaticIPs: []string{"10.244.4.2"},
							}},
						},
						{
							Instances: 2,
							Networks: []destiny.JobNetwork{{
								StaticIPs: []string{"10.244.5.2", "10.244.5.6"},
							}},
						},
					},
				}

				members := manifest.EtcdMembers()
				Expect(members).To(Equal([]destiny.EtcdMember{
					{
						Address: "10.244.4.2",
					},
					{
						Address: "10.244.5.2",
					},
					{
						Address: "10.244.5.6",
					},
				}))
			})
		})

		Context("when the job does not have a network", func() {
			It("returns an empty list", func() {
				manifest := destiny.Manifest{
					Jobs: []destiny.Job{
						{
							Instances: 1,
							Networks:  []destiny.JobNetwork{},
						},
					},
				}

				members := manifest.EtcdMembers()
				Expect(members).To(BeEmpty())
			})
		})

		Context("when the job network does not have enough static IPs", func() {
			It("returns as much about the list as possible", func() {
				manifest := destiny.Manifest{
					Jobs: []destiny.Job{
						{
							Instances: 2,
							Networks: []destiny.JobNetwork{{
								StaticIPs: []string{"10.244.5.2"},
							}},
						},
					},
				}

				members := manifest.EtcdMembers()
				Expect(members).To(Equal([]destiny.EtcdMember{
					{
						Address: "10.244.5.2",
					},
				}))
			})
		})
	})
})
