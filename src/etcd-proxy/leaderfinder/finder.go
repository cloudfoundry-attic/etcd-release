package leaderfinder

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var (
	NoAddressesProvided   = errors.New("no addresses have been provided")
	LeaderNotFound        = errors.New("leader not found")
	MembersNotFound       = errors.New("no etcd members could be found")
	NoClientURLs          = errors.New("no etcd member client urls could be found")
	NoClientURLsForLeader = errors.New("no etcd member client url for leader")
)

type Finder struct {
	addresses []string
	client    getter
}

type members struct {
	Members []member `json:"members"`
}

type member struct {
	ClientURLs []string `json:"clientURLs"`
	ID         string   `json:"id"`
}

type self struct {
	LeaderInfo leaderInfo `json:"leaderInfo"`
}

type leaderInfo struct {
	Leader string `json:"leader"`
}

type getter interface {
	Get(url string) (resp *http.Response, err error)
}

func NewFinder(addresses []string, client getter) Finder {
	return Finder{
		addresses: addresses,
		client:    client,
	}
}

func (f Finder) Find() (string, error) {
	if len(f.addresses) == 0 {
		return "", NoAddressesProvided
	}

	resp, err := f.client.Get(fmt.Sprintf("%s/v2/members", f.addresses[0]))
	if err != nil {
		return "", err
	}

	var members members
	err = json.NewDecoder(resp.Body).Decode(&members)
	if err != nil {
		return "", err
	}

	if len(members.Members) == 0 {
		return "", MembersNotFound
	}

	if len(members.Members[0].ClientURLs) == 0 {
		return "", NoClientURLs
	}

	resp, err = f.client.Get(fmt.Sprintf("%s/v2/stats/self", members.Members[0].ClientURLs[0]))
	if err != nil {
		return "", err
	}

	var self self
	err = json.NewDecoder(resp.Body).Decode(&self)
	if err != nil {
		return "", err
	}

	leaderID := self.LeaderInfo.Leader

	var leaderURL string

	for _, member := range members.Members {
		if member.ID == leaderID {
			if len(member.ClientURLs) == 0 {
				return "", NoClientURLsForLeader
			}

			leaderURL = member.ClientURLs[0]
			break
		}
	}

	if leaderURL == "" {
		return "", LeaderNotFound
	}

	return leaderURL, nil
}
