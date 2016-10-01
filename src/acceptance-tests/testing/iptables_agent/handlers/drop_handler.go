package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type DropHandler struct {
	ipTablesExecutor func([]string) (string, error)
	logger           *log.Logger
}

func NewDropHandler(ipTablesExecutor func([]string) (string, error)) DropHandler {
	return DropHandler{
		ipTablesExecutor: ipTablesExecutor,
		logger:           log.New(os.Stdout, "[drop handler] ", log.LUTC),
	}
}

func (d *DropHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	log.Printf("received request: %s %s\n", req.Method, req.URL.String())
	if req.Method != "PUT" && req.Method != "DELETE" {
		d.logger.Println("error: not a PUT or DELETE request")
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	addr, port, err := d.queryParams(req.URL)
	if err != nil {
		d.logger.Printf("error: missing required params (%s)\n", err.Error())
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(err.Error()))
		return
	}

	command := "-A"
	if req.Method == "DELETE" {
		command = "-D"
	}

	output, err := d.ipTablesExecutor([]string{command, "OUTPUT", "-p", "tcp", "-d", addr, "--dport", port, "-j", "DROP"})
	if err != nil {
		d.logger.Printf("error: iptables failed (%s)\n", err)
		d.logger.Printf("error: iptables failed output: %s\n", string(output))
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(fmt.Sprintf("error: %s\niptables output: %s", err.Error(), string(output))))
		return
	}

	d.logger.Println("request successful")
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
