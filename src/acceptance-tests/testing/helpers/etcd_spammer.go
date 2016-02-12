package helpers

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

func SpamEtcd(done chan struct{}, wg *sync.WaitGroup, machines []string) chan map[string]string {
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

				for _, machine := range machines {
					etcdClient := NewEtcdClient([]string{
						fmt.Sprintf("http://%s:4001", machine),
					})

					err := etcdClient.Set(key, value)
					if err != nil {
						if !strings.Contains(err.Error(), "All the given peers are not reachable") {
							keyVal["error"] = err.Error()
							continue
						}
					} else {
						keyVal[key] = value
					}
				}
			}
		}
	}()

	return keyValChan
}
