package server

import (
	"GO_Plugin/src/config"
	"GO_Plugin/src/plugin/snmp"
	"encoding/json"
	"fmt"
	"github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"
	"strings"
)

const (
	requestType         = "requestType"
	discovery           = "discovery"
	polling             = "polling"
	health_check        = "health_check"
	ok                  = "ok"
	numDiscoveryWorkers = 10
	numPollingWorkers   = 100
)

func StartPull(cfg *config.Config, log *logrus.Logger) {

	pull, err := zmq4.NewSocket(zmq4.PULL)

	if err != nil {

		log.Errorf("Failed to create PULL socket: %v", err)

		return
	}

	defer pull.Close()

	pullAddr := fmt.Sprintf("tcp://%s:%s", cfg.VertxHost, cfg.ZMQPort)

	if err := pull.Connect(pullAddr); err != nil {

		log.Errorf("Failed to connect PULL socket: %v", err)

		return
	}

	push, err := zmq4.NewSocket(zmq4.PUSH)

	if err != nil {

		log.Errorf("Failed to create PUSH socket: %v", err)

		return
	}

	defer push.Close()

	if err := push.Connect(fmt.Sprintf("tcp://%s:%s", cfg.VertxHost, cfg.VertxResponsePort)); err != nil {

		log.Errorf("Failed to connect to Vert.x PULL: %v", err)

		return
	}

	requestDiscoveryChan := make(chan string, 50000)

	requestPollingChan := make(chan string, 50000)

	responseChan := make(chan string, 50000)

	go func() {
		for resp := range responseChan {

			if _, err := push.Send(resp, zmq4.DONTWAIT); err != nil {

				log.Errorf("Failed to send response: %v", err)
			}
		}
	}()

	for i := 0; i < numDiscoveryWorkers; i++ {

		go discoveryWorker(i+1, requestDiscoveryChan, responseChan, log)
	}

	for i := 0; i < numPollingWorkers; i++ {

		go pollingWorker(i+1, requestPollingChan, responseChan, log)
	}

	log.Infof("Server listening on %s...", pullAddr)

	defer close(requestDiscoveryChan)

	defer close(requestPollingChan)

	defer close(responseChan)

	for {
		req, err := pull.Recv(0)

		if err != nil {

			log.Errorf("Pull receive error: %v", err)

			continue
		}

		if req == health_check {

			responseChan <- ok

			continue
		}

		var reqData map[string]interface{}

		_ = json.Unmarshal([]byte(req), &reqData)

		switch strings.ToLower(reqData[requestType].(string)) {

		case discovery:
			requestDiscoveryChan <- req

		case polling:
			requestPollingChan <- req

		default:
			log.Warnf("Unknown requestType: %v", reqData[requestType])
		}
	}
}

func discoveryWorker(id int, reqChan <-chan string, respChan chan<- string, log *logrus.Logger) {

	for req := range reqChan {

		log.Infof("Discovery Worker %d: Processing request", id)

		var reqData map[string]interface{}

		_ = json.Unmarshal([]byte(req), &reqData)

		snmp.Discovery(reqData)

		jsonData, _ := json.Marshal(reqData)

		respChan <- string(jsonData)
	}
}

func pollingWorker(id int, reqChan <-chan string, respChan chan<- string, log *logrus.Logger) {

	for req := range reqChan {

		log.Infof("Polling Worker %d: Processing request", id)

		var reqData map[string]interface{}

		_ = json.Unmarshal([]byte(req), &reqData)

		snmp.FetchSNMPData(reqData)

		jsonData, _ := json.Marshal(reqData)

		respChan <- string(jsonData)
	}
}
