package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type DropHandler struct {
	ipTablesExecutor func([]string) (string, error)
}

func NewDropHandler(ipTablesExecutor func([]string) (string, error)) DropHandler {
	return DropHandler{
		ipTablesExecutor: ipTablesExecutor,
	}
}

func (d *DropHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "PUT" {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	addr, port, err := d.queryParams(req.URL)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(err.Error()))
		return
	}

	output, err := d.ipTablesExecutor([]string{"-A", "OUTPUT", "-p", "tcp", "-d", addr, "--dport", port, "-j", "DROP"})
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf("error: %s\niptables output: %s", err.Error(), string(output))))
		return
	}
}

func (d *DropHandler) queryParams(url *url.URL) (string, string, error) {
	errs := []string{}
	queryVals := url.Query()
	addr := queryVals.Get("addr")
	if addr == "" {
		errs = append(errs, "must provide addr param")
	}

	port := queryVals.Get("port")
	if port == "" {
		errs = append(errs, "must provide port param")
	}

	if len(errs) > 0 {
		return "", "", errors.New(strings.Join(errs, ", "))
	}

	return addr, port, nil
}
