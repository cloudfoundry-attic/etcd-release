package fakes

type Config struct {
	EtcdClientEndpointsCall struct {
		CallCount int
		Returns   struct {
			Endpoints []string
		}
	}
}

func (c *Config) EtcdClientEndpoints() []string {
	c.EtcdClientEndpointsCall.CallCount++

	return c.EtcdClientEndpointsCall.Returns.Endpoints
}
