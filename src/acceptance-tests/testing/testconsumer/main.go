package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/cloudfoundry-incubator/etcd-release/src/acceptance-tests/testing/testconsumer/handlers"
)

type Flags struct {
	Port       string
	EtcdURL    stringSlice
	CACert     string
	ClientCert string
	ClientKey  string
}

type stringSlice []string

func (ss *stringSlice) String() string {
	return fmt.Sprintf("%s", *ss)
}

func (ss *stringSlice) Slice() []string {
	s := []string{}
	for _, v := range *ss {
		s = append(s, v)
	}
	return s
}

func (ss *stringSlice) Set(value string) error {
	*ss = append(*ss, value)

	return nil
}

func main() {
	err := setupStackdriver()
	if err != nil {
		os.Exit(1)
	}
	flags := parseCommandLineFlags()

	kvHandler := handlers.NewKVHandler(flags.EtcdURL.Slice(), flags.CACert, flags.ClientCert, flags.ClientKey)
	leaderNameHandler := handlers.NewLeaderHandler(flags.EtcdURL.Slice()[0], flags.CACert, flags.ClientCert, flags.ClientKey)

	mux := http.NewServeMux()
	mux.HandleFunc("/kv/", func(w http.ResponseWriter, req *http.Request) {
		kvHandler.ServeHTTP(w, req)
	})

	mux.HandleFunc("/leader", func(w http.ResponseWriter, req *http.Request) {
		leaderNameHandler.ServeHTTP(w, req)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf("0.0.0.0:%s", flags.Port), mux))
}

func parseCommandLineFlags() Flags {
	flags := Flags{}
	flag.StringVar(&flags.Port, "port", "", "port to use for test consumer server")
	flag.Var(&flags.EtcdURL, "etcd-service", "url of the etcd service")
	flag.StringVar(&flags.CACert, "ca-cert-file", "", "the file of the CA Certificate")
	flag.StringVar(&flags.ClientCert, "client-ssl-cert-file", "", "the file of the Client SSL Certificate")
	flag.StringVar(&flags.ClientKey, "client-ssl-key-file", "", "the file of the CLient SSL Key")
	flag.Parse()

	return flags
}

func setupStackdriver() error {
	var buf bytes.Buffer

	// curl -sSO https://dl.google.com/cloudagents/install-logging-agent.sh
	cmd := exec.Command("curl", "-sSO", "https://dl.google.com/cloudagents/install-logging-agent.sh")
	cmd.Stdout = &buf
	err := cmd.Start()
	if err != nil {
		fmt.Println("********************************************************************************")
		fmt.Printf("curl error:\n%s", err)
		fmt.Println("********************************************************************************")
		return err
	}
	fmt.Println("********************************************************************************")
	fmt.Printf("curl output:\n%s", string(buf.Bytes()))
	fmt.Println("********************************************************************************")

	// sha256sum install-logging-agent.sh == "8db836510cf65f3fba44a3d49265ed7932e731e7747c6163da1c06bf2063c301  install-logging-agent.sh"
	// cmd = exec.Command("sha256sum", "install-logging-agent.sh")
	// cmd.Stdout = &buf
	// err = cmd.Start()
	// if err != nil {
	// 	fmt.Println("********************************************************************************")
	// 	fmt.Printf("sha256 error:\n%s", err)
	// 	fmt.Println("********************************************************************************")
	// 	return err
	// }
	// sha := string(buf.Bytes())
	// expectedSHA := "8db836510cf65f3fba44a3d49265ed7932e731e7747c6163da1c06bf2063c301  install-logging-agent.sh"
	// if sha != expectedSHA {
	// 	fmt.Println("********************************************************************************")
	// 	fmt.Printf("sha256 for stackdriver logging agent did not match\nGot %s\nExpected %s\n", sha, expectedSHA)
	// 	fmt.Println("********************************************************************************")
	// 	return fmt.Errorf("sha256 for stackdriver logging agent did not match\nGot %s\nExpected %s\n", sha, expectedSHA)
	// }
	// fmt.Println("********************************************************************************")
	// fmt.Printf("sha256 output:\n%s", sha)
	// fmt.Println("********************************************************************************")

	// sudo bash install-logging-agent.sh
	cmd = exec.Command("bash", "install-logging-agent.sh")
	cmd.Stdout = &buf
	err = cmd.Start()
	if err != nil {
		fmt.Println("********************************************************************************")
		fmt.Printf("install-logging-agent error:\n%s", err)
		fmt.Println("********************************************************************************")
		return err
	}
	fmt.Println("********************************************************************************")
	fmt.Printf("install-logging-agent output:\n%s", string(buf.Bytes()))
	fmt.Println("********************************************************************************")

	return nil
}
