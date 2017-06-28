package fakes

import "sync"

type atomicCount struct {
	value int
	sync.Mutex
}

func (c *atomicCount) inc() {
	c.Lock()
	defer c.Unlock()

	c.value++
}

func (c *atomicCount) Value() int {
	c.Lock()
	defer c.Unlock()

	return c.value
}

type KV struct {
	KeyVals map[string]string

	AddressCall struct {
		Returns struct {
			Address string
		}
	}

	SetCall struct {
		CallCount *atomicCount
		Stub      func(string, string) error
		Receives  struct {
			Key   string
			Value string
		}
		Returns struct {
			Error error
		}
	}

	GetCall struct {
		CallCount int
		Stub      func(string) (string, error)
		Receives  struct {
			Key string
		}
		Returns struct {
			Value string
			Error error
		}
	}
}

func NewKV() *KV {
	kv := &KV{KeyVals: map[string]string{}}
	kv.SetCall.CallCount = &atomicCount{}
	return kv
}

func (k *KV) Set(key, value string) error {
	k.SetCall.CallCount.inc()
	k.SetCall.Receives.Key = key
	k.SetCall.Receives.Value = value

	k.KeyVals[key] = value

	if k.SetCall.Stub != nil {
		return k.SetCall.Stub(key, value)
	}

	return k.SetCall.Returns.Error
}

func (k *KV) Get(key string) (string, error) {
	k.GetCall.CallCount++
	k.GetCall.Receives.Key = key

	if k.GetCall.Stub != nil {
		return k.GetCall.Stub(key)
	}

	return k.KeyVals[key], k.GetCall.Returns.Error
}

func (k *KV) Address() string {
	return k.AddressCall.Returns.Address
}
