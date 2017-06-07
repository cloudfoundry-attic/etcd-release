package fakes

type SyncController struct {
	VerifySyncedCall struct {
		CallCount int
		Returns   struct {
			Error error
		}
	}
}

func (s *SyncController) VerifySynced() error {
	s.VerifySyncedCall.CallCount++
	return s.VerifySyncedCall.Returns.Error
}
