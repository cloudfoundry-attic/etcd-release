package main

import (
	"acceptance-tests/testing/iptables_agent/handlers"
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
)

type Flags struct {
	Port string
}

func main() {
	flags := parseCommandLineFlags()

	dropHandler := handlers.NewDropHandler(func(args []string) (string, error) {
		cmd := exec.Command("iptables", args...)
		output := bytes.NewBuffer([]byte{})
		cmd.Stdout = output
		cmd.Stderr = output
		err := cmd.Run()
		if err != nil {
			return output.String(), err
		}

		return "", nil
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/drop", func(w http.ResponseWriter, req *http.Request) {
		dropHandler.ServeHTTP(w, req)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", flags.Port), mux))
}

func parseCommandLineFlags() Flags {
	flags := Flags{}
	flag.StringVar(&flags.Port, "port", "", "port to use for iptables agent server")
	flag.Parse()

	return flags
}
