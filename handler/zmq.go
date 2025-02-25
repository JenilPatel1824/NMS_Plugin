package handler

import (
	"GO_Plugin/config"
	"GO_Plugin/snmp"
	"encoding/json"
	"fmt"
	"github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	routerAddrFormat  = "tcp://*:%s"
	dealerAddr        = "inproc://workers"
	numWorkers        = 10
	errorKey          = "error"
	detailKey         = "details"
	invalidRequest    = "Invalid request format"
	ipKey             = "ip"
	communityKey      = "community"
	versionKey        = "version"
	ErrorMissingField = "Missing or null field"
)

// StartZMQRouter initializes and starts a ZeroMQ Router-Dealer pattern for handling client requests and worker responses.
// It sets up a ROUTER socket for client communication and a DEALER socket for worker load balancing.
// The function binds sockets, starts worker goroutines, and runs a proxy to manage request-response routing.
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

	dealerAddr := dealerAddr

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

// startWorker initializes a worker in a ZeroMQ Dealer-Worker pattern for processing SNMP requests and sending responses.
// dealerAddr specifies the DEALER socket address the worker connects to.
// workerID is a unique identifier for the worker instance.
// log is the logging instance used to log worker operations and errors.
// wg is a wait group that ensures coordinated goroutine execution completion.
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

		log.Infof("Worker %d: Received request: %s", workerID, req)

		startTime := time.Now()

		log.Infof("Worker %d: Start time after receiving request: %v", workerID, startTime)

		var reqData map[string]string

		if err := json.Unmarshal([]byte(req), &reqData); err != nil {

			errorResponse := map[string]string{

				errorKey: invalidRequest,

				detailKey: err.Error(),
			}

			errorJSON, _ := json.Marshal(errorResponse)

			_, err := worker.Send(string(errorJSON), 0)

			if err != nil {

				continue
			}

			continue
		}

		_, missingFields := validateRequest(reqData)

		if len(missingFields) > 0 {

			response := map[string]interface{}{

				errorKey: ErrorMissingField,

				detailKey: missingFields,
			}

			responseData, _ := json.Marshal(response)

			_, err2 := worker.Send(string(responseData), 0)

			if err2 != nil {

				log.Errorf("Worker %d: Failed to send response: %v", workerID, err2)

				continue
			}

			continue
		}

		ip := reqData[ipKey]

		community := reqData[communityKey]

		version := reqData[versionKey]

		result := snmp.FetchSNMPData(ip, community, version)

		jsonData, err := json.Marshal(result)

		if err != nil {

			log.Errorf("Worker %d: Failed to marshal JSON: %v", workerID, err)

			continue
		}

		log.Infof("Worker %d: Sending processed response: ", workerID)

		_, err = worker.Send(string(jsonData), 0)

		if err != nil {

			log.Errorf("Worker %d: Failed to send response: %v", workerID, err)

			continue

		} else {

			log.Infof("Worker %d: Sent response: ", workerID)
		}

		endTime := time.Now()

		log.Infof("Worker %d: End time after sending response: %v", workerID, endTime)

		timeTaken := endTime.Sub(startTime)

		log.Infof("Worker %d: Time taken for this request: %v", workerID, timeTaken)
	}
}

// validateRequest checks the presence of required fields in the request data and identifies missing or empty fields.
// It returns a map of valid fields with their corresponding values, and a map of missing fields with error messages.
func validateRequest(reqData map[string]string) (map[string]string, map[string]string) {

	missingFields := make(map[string]string)

	values := make(map[string]string)

	for _, field := range []string{ipKey, communityKey, versionKey} {

		value, ok := reqData[field]

		if !ok || value == "" {

			missingFields[field] = ErrorMissingField

		} else {

			values[field] = value
		}
	}

	return values, missingFields
}
