package client

import (
	"bytes"
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

	_, err = http.Post(c.baseURL+"/api/v1/incidents", "text/json", bytes.NewBuffer(jsonCommand))
	if err != nil {
		return err
	}

	return nil

}
