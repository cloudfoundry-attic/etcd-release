package main

import (
	"acceptance-tests/testing/testconsumer/handlers"
	"flag"
	"fmt"
	"log"
	"net/http"
)

type Flags struct {
	Port       string
	EtcdURL    string
	CACert     string
	ClientCert string
	ClientKey  string
}

func main() {
	flags := parseCommandLineFlags()

	kvHandler := handlers.NewKVHandler(flags.EtcdURL, flags.CACert, flags.ClientCert, flags.ClientKey)

	mux := http.NewServeMux()
	mux.HandleFunc("/kv/", func(w http.ResponseWriter, req *http.Request) {
		kvHandler.ServeHTTP(w, req)
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", flags.Port), mux))
}

func parseCommandLineFlags() Flags {
	flags := Flags{}
	flag.StringVar(&flags.Port, "port", "", "port to use for test consumer server")
	flag.StringVar(&flags.EtcdURL, "etcd-service", "", "url of the etcd service")
	flag.StringVar(&flags.CACert, "ca-cert-file", "", "the file of the CA Certificate")
	flag.StringVar(&flags.ClientCert, "client-ssl-cert-file", "", "the file of the Client SSL Certificate")
	flag.StringVar(&flags.ClientKey, "client-ssl-key-file", "", "the file of the CLient SSL Key")
	flag.Parse()

	return flags
}
