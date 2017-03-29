package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	lainlet "github.com/laincloud/lainlet/client"
	"os/exec"
	"time"
)

type watcherCallback func(data interface{})

func ExecCommand(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	var cmdOut bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdOut
	err := cmd.Run()
	return cmdOut.String(), err
}

func WatchConfig(log *logrus.Logger, lainlet *lainlet.Client, configKeyPrefix string, watchCh <-chan struct{}, callback watcherCallback) {
	url := fmt.Sprintf("/v2/configwatcher?target=%s&heartbeat=5", configKeyPrefix)
	retryCounter := 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		breakWatch := false
		ch, err := lainlet.Watch(url, ctx)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":          err,
				"retryCounter": retryCounter,
			}).Error("Fail to connect lainlet")
			if retryCounter > 3 {
				time.Sleep(30 * time.Second)
			} else {
				time.Sleep(1 * time.Second)
			}
			retryCounter++
			continue
		}
		retryCounter = 0
		for {
			select {
			case event, ok := <-ch:
				if !ok {
					breakWatch = true
					break
				}
				if event.Id == 0 {
					// lainlet error for etcd down
					if event.Event == "error" {
						log.WithFields(logrus.Fields{
							"id":    event.Id,
							"event": event.Event,
						}).Error("Fail to watch lainlet")
						time.Sleep(5 * time.Second)
					}
					continue
				}
				var addrs interface{}
				err = json.Unmarshal(event.Data, &addrs)
				callback(addrs)
			case <-watchCh:
				return
			}
			if breakWatch {
				break
			}
		}
		log.Error("Fail to watch lainlet")
	}
}