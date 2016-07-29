package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

var (
	OutputPath         string
	RecentLogsPath     string
	SyslogListenerPort string
)

func main() {
	f, err := os.Open(OutputPath)
	if err != nil {
		panic(err)
	}

	defer f.Close()

	var commands [][]string
	if err := json.NewDecoder(f).Decode(&commands); err != nil {
		panic(err)
	}

	commands = append(commands, os.Args[1:])

	buf, err := json.Marshal(commands)
	if err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(OutputPath, buf, os.ModePerm); err != nil {
		panic(err)
	}

	if os.Args[1] == "logs" {
		fmt.Printf("2016-07-28T17:02:49.24-0700 [App/0]      OUT ADDRESS: |127.0.0.1:%s|\n", SyslogListenerPort)

		logs, err := ioutil.ReadFile(RecentLogsPath)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(logs))
	}
}
