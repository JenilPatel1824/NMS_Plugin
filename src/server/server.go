package server

import (
	"GO_Plugin/src/config"
	"GO_Plugin/src/plugin/snmp"
	"encoding/json"
	"fmt"
	"github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
)

const (
	routerAddrFormat = "tcp://*:%s"
	dealerAddr       = "inproc://workers"
	numWorkers       = 5
	errors           = "error"
	details          = "details"
	invalidRequest   = "Invalid request format"
	requestType      = "requestType"
	discovery        = "discovery"
	polling          = "polling"
	requestID        = "request_id"
	status           = "status"
	fail             = "fail"
)

// StartZMQRouter initializes and starts a ZeroMQ Router-Dealer pattern for handling client requests and worker responses.
func StartZMQRouter(cfg *config.Config, log *logrus.Logger) {

	router, err := zmq4.NewSocket(zmq4.ROUTER)

	if err != nil {

		log.Errorf("Failed to create ROUTER socket: %v", err)

		return
	}

	defer router.Close()

	dealer, err := zmq4.NewSocket(zmq4.DEALER)

	if err != nil {

		log.Errorf("Failed to create DEALER socket: %v", err)

		return
	}

	defer dealer.Close()

	routerAddr := fmt.Sprintf(routerAddrFormat, cfg.ZMQPort)

	if err := router.Bind(routerAddr); err != nil {

		log.Errorf("Failed to bind ROUTER socket: %v", err)

		return
	}

	if err := dealer.Bind(dealerAddr); err != nil {

		log.Errorf("Failed to bind DEALER socket: %v", err)

		return
	}

	log.Infof("Polling engine listening on %s...", routerAddr)

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {

		wg.Add(1)

		go startWorker(dealerAddr, i+1, log, &wg)
	}

	err = zmq4.Proxy(router, dealer, nil)

	if err != nil {

		log.Errorf("ZeroMQ Proxy error: %v", err)
	}

	wg.Wait()
}

// startWorker initializes a worker for processing SNMP requests.
func startWorker(dealerAddr string, workerID int, log *logrus.Logger, wg *sync.WaitGroup) {

	defer wg.Done()

	worker, err := zmq4.NewSocket(zmq4.REP)

	if err != nil {

		log.Errorf("Worker %d: Failed to create socket: %v", workerID, err)

		return
	}

	defer worker.Close()

	err = worker.Connect(dealerAddr)

	if err != nil {

		log.Errorf("Worker %d: Failed to connect to DEALER: %v", workerID, err)

		return
	}

	log.Infof("Worker %d: Ready and connected to DEALER", workerID)

	for {

		req, err := worker.Recv(0)

		if err != nil {

			log.Errorf("Worker %d: Failed to receive message: %v", workerID, err)

			continue
		}

		if req == "health_check" {

			worker.Send("ok", 0)

			continue
		}

		log.Infof("Worker %d: Received request: %s", workerID, req)

		var reqData map[string]interface{}

		if err := json.Unmarshal([]byte(req), &reqData); err != nil {

			errorResponse := map[string]string{

				requestID: reqData[requestID].(string),

				errors: invalidRequest,

				details: err.Error(),

				status: fail,
			}

			errorJSON, _ := json.Marshal(errorResponse)

			worker.Send(string(errorJSON), 0)

			continue
		}

		requestType := reqData[requestType]

		switch strings.ToLower(requestType.(string)) {

		case discovery:

			log.Infof("Worker %d: Processing discovery request: ", workerID)

			snmp.Discovery(reqData)

			jsonResponse, _ := json.Marshal(reqData)

			worker.Send(string(jsonResponse), 0)

		case polling:

			log.Infof("Worker %d: Processing polling request: ", workerID)

			snmp.FetchSNMPData(reqData)

			jsonData, _ := json.Marshal(reqData)

			log.Infof("Worker %d: sending back response: ", workerID)

			worker.Send(string(jsonData), 0)

		default:

			reqData[errors] = "request type not supported"

			reqData[status] = fail

			jsonResponse, _ := json.Marshal(reqData)

			log.Infof("Worker %d: sending response: ", workerID)

			worker.Send(string(jsonResponse), 0)
		}
	}
}
