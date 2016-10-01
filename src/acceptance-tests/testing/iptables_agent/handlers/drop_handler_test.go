package handlers_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	"acceptance-tests/testing/iptables_agent/handlers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("DropHandler", func() {
	var (
		dropHandler        handlers.DropHandler
		iptables           func(args []string) (string, error)
		actualIPTablesArgs []string
	)

	BeforeEach(func() {
		actualIPTablesArgs = nil

		iptables = func(args []string) (string, error) {
			actualIPTablesArgs = args
			return "", nil
		}

		dropHandler = handlers.NewDropHandler(iptables)
	})

	DescribeTable("required params", func(requestParams, expectedErrMsg string) {
		url := fmt.Sprintf("/drop?%s", requestParams)
		request, err := http.NewRequest("PUT", url, strings.NewReader(""))
		Expect(err).NotTo(HaveOccurred())

		recorder := httptest.NewRecorder()

		dropHandler.ServeHTTP(recorder, request)

		Expect(recorder.Code).To(Equal(http.StatusBadRequest))

		respContents, err := ioutil.ReadAll(recorder.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(respContents)).To(Equal(expectedErrMsg))
	},
		Entry("missing addr", "port=9999", "must provide addr param"),
		Entry("missing port", "addr=some-addr-ip", "must provide port param"),
		Entry("missing addr and port", "", "must provide addr param, must provide port param"),
	)

	Context("failure cases", func() {
		It("returns a bad request when the request is not a PUT or a DELETE", func() {
			request, err := http.NewRequest("GET", "/drop", strings.NewReader(""))
			Expect(err).NotTo(HaveOccurred())

			recorder := httptest.NewRecorder()

			dropHandler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusNotFound))
		})

		It("returns an error when ipTablesExecutor fails", func() {
			iptables = func(_ []string) (string, error) {
				return "some error occurred", errors.New("ipTablesExecutor failed")
			}
			dropHandler = handlers.NewDropHandler(iptables)

			request, err := http.NewRequest("PUT", "/drop?addr=some-ip-addr&port=9999", strings.NewReader(""))
			Expect(err).NotTo(HaveOccurred())

			recorder := httptest.NewRecorder()

			dropHandler.ServeHTTP(recorder, request)

			Expect(recorder.Code).To(Equal(http.StatusInternalServerError))

			respContents, err := ioutil.ReadAll(recorder.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(respContents)).To(Equal("error: ipTablesExecutor failed\niptables output: some error occurred"))
		})
	})

	Context("when adding a drop rule with PUT request", func() {
		It("calls iptables with the correct arguments", func() {
			request, err := http.NewRequest("PUT", "/drop?addr=some-ip-addr&port=9898", strings.NewReader(""))
			Expect(err).NotTo(HaveOccurred())

			recorder := httptest.NewRecorder()

			dropHandler.ServeHTTP(recorder, request)

			Expect(actualIPTablesArgs).To(Equal([]string{
				"-A", "OUTPUT",
				"-p", "tcp",
				"-d", "some-ip-addr",
				"--dport", "9898",
				"-j", "DROP",
			}))
			Expect(recorder.Code).To(Equal(http.StatusOK))
		})
	})

	Context("when removing a drop rule with DELETE request", func() {
		It("calls iptables with the correct arguments", func() {
			request, err := http.NewRequest("DELETE", "/drop?addr=some-ip-addr&port=9898", strings.NewReader(""))
			Expect(err).NotTo(HaveOccurred())

			recorder := httptest.NewRecorder()

			dropHandler.ServeHTTP(recorder, request)
			Expect(actualIPTablesArgs).To(Equal([]string{
				"-D", "OUTPUT",
				"-p", "tcp",
				"-d", "some-ip-addr",
				"--dport", "9898",
				"-j", "DROP",
			}))
			Expect(recorder.Code).To(Equal(http.StatusOK))
		})
	})
})
