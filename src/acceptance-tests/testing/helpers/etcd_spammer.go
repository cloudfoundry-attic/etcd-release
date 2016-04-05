package helpers

import (
	"acceptance-tests/testing/etcd"

	"fmt"
	"strings"
	"sync"
	"time"
)

func SpamEtcd(done chan struct{}, wg *sync.WaitGroup, etcdClient etcd.Client) chan map[string]string {
	keyValChan := make(chan map[string]string, 1)
	wg.Add(1)

	go func() {
		keyVal := make(map[string]string)
		for {
			select {
			case <-done:
				keyValChan <- keyVal

				wg.Done()
				return
			case <-time.After(1 * time.Second):
				guid, err := NewGUID()
				if err != nil {
					keyVal["error"] = err.Error()
					continue
				}

				key := fmt.Sprintf("etcd-key-%s", guid)
				value := fmt.Sprintf("etcd-value-%s", guid)

				err = etcdClient.Set(key, value)
				if err != nil {
					switch {
					case strings.Contains(err.Error(), "All the given peers are not reachable"):
						continue
					case strings.Contains(err.Error(), "dial tcp 10.244.16.10:6769: getsockopt: connection refused"):
						continue
					case strings.Contains(err.Error(), "EOF"):
						continue
					default:
						keyVal["error"] = err.Error()
						continue
					}
				} else {
					keyVal[key] = value
				}
			}
		}
	}()

	return keyValChan
}
