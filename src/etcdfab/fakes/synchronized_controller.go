package fakes

type SynchronizedController struct {
	VerifySyncedCall struct {
		CallCount int
		Returns   struct {
			Error error
		}
	}
}

func (s *SynchronizedController) VerifySynced() error {
	s.VerifySyncedCall.CallCount++
	return s.VerifySyncedCall.Returns.Error
}
