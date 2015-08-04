package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
)

type Client struct {
	baseURL string
}

type deployment struct {
	Name string
	Jobs []job
}

type job struct {
	Name    string
	Indices []int
}

type killTask struct {
	Type string
}

type killCommand struct {
	Tasks       []interface{}
	Deployments []deployment
}

func NewClient(baseURL string) Client {
	return Client{
		baseURL: baseURL,
	}
}

func (c Client) KillIndices(deploymentName, jobName string, indices []int) error {
	command := killCommand{
		Tasks: []interface{}{
			killTask{Type: "kill"},
		},
		Deployments: []deployment{{
			Name: deploymentName,
			Jobs: []job{{Name: jobName, Indices: indices}},
		}},
	}

	jsonCommand, err := json.Marshal(command)
	if err != nil {
		return err
	}

	request, err := http.NewRequest("POST", c.baseURL+"/api/v1/incidents", bytes.NewBuffer(jsonCommand))
	if err != nil {
		return err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	_, err = client.Do(request)
	if err != nil {
		return err
	}

	return nil
}
