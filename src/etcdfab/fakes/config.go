package fakes

type Config struct {
	EtcdClientSelfEndpointCall struct {
		CallCount int
		Returns   struct {
			Endpoint string
		}
	}
	EtcdClientEndpointsCall struct {
		CallCount int
		Returns   struct {
			Endpoints []string
		}
	}
	RequireSSLCall struct {
		CallCount int
		Returns   struct {
			RequireSSL bool
		}
	}
	CertDirCall struct {
		CallCount int
		Returns   struct {
			CertDir string
		}
	}
}

func (c *Config) EtcdClientSelfEndpoint() string {
	c.EtcdClientSelfEndpointCall.CallCount++

	return c.EtcdClientSelfEndpointCall.Returns.Endpoint
}

func (c *Config) EtcdClientEndpoints() []string {
	c.EtcdClientEndpointsCall.CallCount++

	return c.EtcdClientEndpointsCall.Returns.Endpoints
}

func (c *Config) RequireSSL() bool {
	c.RequireSSLCall.CallCount++

	return c.RequireSSLCall.Returns.RequireSSL
}

func (c *Config) CertDir() string {
	c.CertDirCall.CallCount++

	return c.CertDirCall.Returns.CertDir
}
