package helpers_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-incubator/etcd-release/src/acceptance-tests/testing/helpers"
	"github.com/cloudfoundry-incubator/etcd-release/src/acceptance-tests/testing/helpers/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Spammer", func() {
	var (
		kv            *fakes.KV
		spammer       *helpers.Spammer
		spammerPrefix string
	)

	BeforeEach(func() {
		kv = fakes.NewKV()
		kv.AddressCall.Returns.Address = "http://some-address"

		spammerPrefix = "some-prefix"

		spammer = helpers.NewSpammer(kv, time.Duration(0), spammerPrefix)
	})

	Describe("Check", func() {
		It("gets all the sets", func() {
			spammerRun(kv, spammer, 1)

			Expect(spammer.Check()).To(Succeed())
			Expect(kv.GetCall.CallCount).Should(Equal(kv.SetCall.CallCount.Value()))
		})

		It("returns an error when a key doesn't exist", func() {
			kv.GetCall.Returns.Error = errors.New("could not find key: some-prefix-some-key-0")

			spammerRun(kv, spammer, 1)

			err := spammer.Check()
			Expect(err).To(MatchError(ContainSubstring("could not find key: some-prefix-some-key-0")))
		})

		It("returns an error when a key doesn't match it's value", func() {
			spammerRun(kv, spammer, 1)

			Expect(kv.KeyVals).To(HaveKeyWithValue("some-prefix-some-key-0", "some-prefix-some-value-0"))
			kv.KeyVals["some-prefix-some-key-0"] = "banana"

			err := spammer.Check()
			Expect(err).To(MatchError(ContainSubstring("value for key \"some-prefix-some-key-0\" does not match: expected \"some-prefix-some-value-0\", got \"banana\"")))
		})

		Context("when an error occurs", func() {
			It("returns an error if no keys were written", func() {
				kv.SetCall.Returns.Error = errors.New("dial tcp some-address: getsockopt: connection refused")

				spammerRun(kv, spammer, 1)

				err := spammer.Check()
				Expect(err).To(MatchError(ContainSubstring("0 keys have been written")))
			})

			It("does not return an error when an underlying etcd api client error occurs", func() {
				kv.SetCall.Stub = func(string, string) error {
					if kv.SetCall.CallCount.Value() < 10 {
						return errors.New("dial tcp some-address: getsockopt: connection refused")
					}
					return nil
				}

				spammerRun(kv, spammer, 100)

				err := spammer.Check()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("FailPercentages", func() {
		Context("when nothing is lost", func() {
			It("returns the failure percentages for reads/writes", func() {
				spammerRun(kv, spammer, 1)

				read, write := spammer.FailPercentages()
				spammer.Check()

				Expect(read).To(Equal(0))
				Expect(write).To(Equal(0))
			})
		})

		Context("when data is lost", func() {
			Context("when data fails to be read", func() {
				It("returns the failure percentages for reads/writes", func() {
					kv.GetCall.Stub = func(key string) (string, error) {
						if kv.GetCall.CallCount < 200 {
							return "", errors.New("could not find key: some-prefix-some-key-0")
						}

						return kv.KeyVals[key], nil
					}

					spammerRun(kv, spammer, 3)
					spammer.Check()

					read, write := spammer.FailPercentages()
					Expect(read).To(BeNumerically("~", 35, 4))
					Expect(write).To(Equal(0))
				})
			})

			Context("when the data fails to write", func() {
				It("returns the failure percentages for reads/writes", func() {
					kv.SetCall.Stub = func(key, value string) error {
						if kv.SetCall.CallCount.Value() < 200 {
							return errors.New("could not find key: some-prefix-some-key-0")
						}

						return nil
					}

					spammerRun(kv, spammer, 3)
					spammer.Check()

					read, write := spammer.FailPercentages()
					Expect(read).To(Equal(0))
					Expect(write).To(BeNumerically("~", 35, 4))
				})
			})
		})
	})
})

func spammerRun(kv *fakes.KV, spammer *helpers.Spammer, count int) {
	spammer.Spam()

	Eventually(func() int {
		return kv.SetCall.CallCount.Value()
	}).Should(BeNumerically(">", count))

	spammer.Stop()
}
