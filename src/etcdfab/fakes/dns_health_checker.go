package fakes

type DNSHealthChecker struct {
	CheckARecordCall struct {
		CallCount int
		Receives  struct {
			Hostname string
		}
		Returns struct {
			Error error
		}
	}
}

func (d *DNSHealthChecker) CheckARecord(hostname string) error {
	d.CheckARecordCall.CallCount++
	d.CheckARecordCall.Receives.Hostname = hostname
	return d.CheckARecordCall.Returns.Error
}
