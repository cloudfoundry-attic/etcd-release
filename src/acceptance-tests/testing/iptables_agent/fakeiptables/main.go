package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

var (
	SinkURL string
	Fail    string
)

func main() {
	if Fail == "true" {
		fmt.Print("fast failing...")
		os.Exit(1)
	}

	reqBody := strings.Join(os.Args, " ")
	req, err := http.NewRequest("PUT", SinkURL, strings.NewReader(reqBody))
	if err != nil {
		panic(err)
	}

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
}
