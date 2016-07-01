package leaderfinder_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"etcd-proxy/leaderfinder"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeGetter struct {
	GetCall struct {
		CallCount int
	}
}

func (g *fakeGetter) Get(url string) (resp *http.Response, err error) {
	g.GetCall.CallCount++

	return http.Get(url)
}

var _ = Describe("Finder", func() {
	var (
		getter *fakeGetter
	)

	BeforeEach(func() {
		getter = &fakeGetter{}
	})

	Describe("Find", func() {
		It("finds the the leader in an etcd cluster", func() {
			var (
				node1Server *httptest.Server
				node2Server *httptest.Server
				node3Server *httptest.Server
			)

			node1Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/members":
					w.Write([]byte(fmt.Sprintf(`{
					  "members": [
						{
						  "clientURLs": [
						  %q
						  ],
						  "name": "etcd-z1-0",
						  "id": "1b8722e8a026db8e"
						},
						{
						  "clientURLs": [
						  %q
						  ],
						  "name": "etcd-z1-1",
						  "id": "2ff908d1599e9e72"
						},
						{
						  "clientURLs": [
						  %q
						  ],
						  "name": "etcd-z1-2",
						  "id": "7be499c93624e6d5"
						}
					  ]
					}`, node1Server.URL, node2Server.URL, node3Server.URL)))
					return
				case "/v2/stats/self":
					w.Write([]byte(`{
					  "name": "etcd-z1-0",
					  "id": "1b8722e8a026db8e",
					  "state": "StateFollower",
					  "leaderInfo": {
						"leader": "2ff908d1599e9e72"
					  }
					}`))
					return
				}

				w.WriteHeader(http.StatusTeapot)
			}))

			node2Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/members":
					w.Write([]byte(fmt.Sprintf(`{
					  "members": [
						{
						  "clientURLs": [
						  %q
						  ],
						  "name": "etcd-z1-0",
						  "id": "1b8722e8a026db8e"
						},
						{
						  "clientURLs": [
						  %q
						  ],
						  "name": "etcd-z1-1",
						  "id": "2ff908d1599e9e72"
						},
						{
						  "clientURLs": [
						  %q
						  ],
						  "name": "etcd-z1-2",
						  "id": "7be499c93624e6d5"
						}
					  ]
					}`, node1Server.URL, node2Server.URL, node3Server.URL)))
					return
				case "/v2/stats/self":
					w.Write([]byte(`{
					  "name": "etcd-z1-1",
					  "id": "2ff908d1599e9e72",
					  "state": "StateLeader",
					  "leaderInfo": {
						"leader": "2ff908d1599e9e72"
					  }
					}`))
					return
				}

				w.WriteHeader(http.StatusTeapot)
			}))

			node3Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/members":
					w.Write([]byte(fmt.Sprintf(`{
					  "members": [
						{
						  "clientURLs": [
						  %q
						  ],
						  "name": "etcd-z1-0",
						  "id": "1b8722e8a026db8e"
						},
						{
						  "clientURLs": [
						  %q
						  ],
						  "name": "etcd-z1-1",
						  "id": "2ff908d1599e9e72"
						},
						{
						  "clientURLs": [
						  %q
						  ],
						  "name": "etcd-z1-2",
						  "id": "7be499c93624e6d5"
						}
					  ]
					}`, node1Server.URL, node2Server.URL, node3Server.URL)))
					return
				case "/v2/stats/self":
					w.Write([]byte(`{
					  "name": "etcd-z1-2",
					  "id": "7be499c93624e6d5",
					  "state": "StateFollower",
					  "leaderInfo": {
						"leader": "2ff908d1599e9e72"
					  }
					}`))
					return
				}

				w.WriteHeader(http.StatusTeapot)
			}))

			finder := leaderfinder.NewFinder([]string{node1Server.URL, node2Server.URL, node3Server.URL}, getter)

			leader, err := finder.Find()
			Expect(err).NotTo(HaveOccurred())

			Expect(leader).To(Equal(node2Server.URL))
			Expect(getter.GetCall.CallCount).To(Equal(2))
		})

		Context("failure cases", func() {
			It("returns an address if no addresses have been provided", func() {
				finder := leaderfinder.NewFinder([]string{}, getter)

				_, err := finder.Find()
				Expect(err).To(MatchError("no addresses have been provided"))
			})

			It("returns an error when the call to /v2/members fails", func() {
				finder := leaderfinder.NewFinder([]string{"%%%%%%%"}, getter)

				_, err := finder.Find()
				Expect(err).To(MatchError(ContainSubstring("invalid URL escape \"%%%\"")))
			})

			It("returns an error when the call to /v2/members returns malformed json", func() {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte(`%%%%%%%`))
				}))
				finder := leaderfinder.NewFinder([]string{server.URL}, getter)

				_, err := finder.Find()
				Expect(err).To(MatchError("invalid character '%' looking for beginning of value"))
			})

			It("returns an error when no members have been found", func() {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/v2/members":
						w.Write([]byte(`{
						  "members": []
						}`))
						return
					}

					w.WriteHeader(http.StatusTeapot)
				}))

				finder := leaderfinder.NewFinder([]string{server.URL}, getter)

				_, err := finder.Find()
				Expect(err).To(MatchError(leaderfinder.MembersNotFound))
			})

			It("returns an error when no member clientURLs have been found", func() {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/v2/members":
						w.Write([]byte(`{
						  "members": [
							{
							  "name": "etcd-z1-0",
							  "id": "1b8722e8a026db8e"
							}
						  ]
						}`))
						return
					}

					w.WriteHeader(http.StatusTeapot)
				}))

				finder := leaderfinder.NewFinder([]string{server.URL}, getter)

				_, err := finder.Find()
				Expect(err).To(MatchError(leaderfinder.NoClientURLs))
			})

			It("returns an error when the call to /v2/stats/self fails", func() {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/v2/members":
						w.Write([]byte(`{
						  "members": [
							{
							  "clientURLs": [
							  	"%%%%%%%%%%"
							  ],
							  "name": "etcd-z1-0",
							  "id": "1b8722e8a026db8e"
							}
						  ]
						}`))
						return
					}

					w.WriteHeader(http.StatusTeapot)
				}))

				finder := leaderfinder.NewFinder([]string{server.URL}, getter)

				_, err := finder.Find()
				Expect(err).To(MatchError(ContainSubstring("invalid URL escape \"%%%\"")))
			})

			It("returns an error when the call to /v2/stats/self returns malformed json", func() {
				var server *httptest.Server

				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/v2/members":
						w.Write([]byte(fmt.Sprintf(`{
						  "members": [
							{
							  "clientURLs": [
							  	%q
							  ],
							  "name": "etcd-z1-0",
							  "id": "1b8722e8a026db8e"
							}
						  ]
						}`, server.URL)))
						return
					case "/v2/stats/self":
						w.Write([]byte(`%%%%%`))
						return
					}

					w.WriteHeader(http.StatusTeapot)
				}))

				finder := leaderfinder.NewFinder([]string{server.URL}, getter)

				_, err := finder.Find()
				Expect(err).To(MatchError("invalid character '%' looking for beginning of value"))
			})

			It("returns an error if the leader does not have a client url", func() {
				var server *httptest.Server
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/v2/members":
						w.Write([]byte(fmt.Sprintf(`{
						  "members": [
							{
							  "clientURLs": [
							  	%q
							  ],
							  "name": "etcd-z1-1",
							  "id": "1b8722e8a026db8e"
							},
							{
							  "clientURLs": [],
							  "name": "etcd-z1-0",
							  "id": "2ff908d1599e9e72"
							}
						  ]
						}`, server.URL)))
						return
					case "/v2/stats/self":
						w.Write([]byte(`{
						  "name": "etcd-z1-0",
						  "id": "1b8722e8a026db8e",
						  "state": "StateFollower",
						  "leaderInfo": {
							"leader": "2ff908d1599e9e72"
						  }
						}`))
						return
					}

					w.WriteHeader(http.StatusTeapot)
				}))

				finder := leaderfinder.NewFinder([]string{server.URL}, getter)

				_, err := finder.Find()
				Expect(err).To(MatchError(leaderfinder.NoClientURLsForLeader))
			})

			It("returns a LeaderNotFound error when a leader cannot be found", func() {
				var server *httptest.Server

				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/v2/members":
						w.Write([]byte(fmt.Sprintf(`{
						  "members": [
							{
							  "clientURLs": [
							  	%q
							  ],
							  "name": "etcd-z1-0",
							  "id": "1b8722e8a026db8e"
							}
						  ]
						}`, server.URL)))
						return
					case "/v2/stats/self":
						w.Write([]byte(`{
						  "name": "etcd-z1-0",
						  "id": "1b8722e8a026db8e",
						  "state": "StateFollower",
						  "leaderInfo": {
							"leader": "2ff908d1599e9e72"
						  }
						}`))
						return
					}

					w.WriteHeader(http.StatusTeapot)
				}))

				finder := leaderfinder.NewFinder([]string{server.URL}, getter)

				_, err := finder.Find()
				Expect(err).To(MatchError(leaderfinder.LeaderNotFound))
			})
		})
	})
})
