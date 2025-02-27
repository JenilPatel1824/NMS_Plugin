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
	routerAddrFormat  = "tcp://*:%s"
	dealerAddr        = "inproc://workers"
	numWorkers        = 5
	errorKey          = "error"
	detailKey         = "details"
	invalidRequest    = "Invalid request format"
	ipKey             = "ip"
	communityKey      = "community"
	versionKey        = "version"
	errorMissingField = "Missing or null field"
	pluginTypeKey     = "pluginType"
	requestTypeKey    = "requestType"
	snmpKey           = "snmp"
	discoveryKey      = "discovery"
	pollingKey        = "pollingKey"
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

				errorKey: errorMissingField,

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

		pluginType := reqData[pluginTypeKey]

		requestType := reqData[requestTypeKey]

		if strings.ToLower(pluginType) != snmpKey {

			response := map[string]interface{}{

				"status": "fail",

				"message": "System type not supported",
			}

			jsonResponse, _ := json.Marshal(response)

			worker.Send(string(jsonResponse), 0)

			continue
		}

		switch strings.ToLower(requestType) {

		case discoveryKey:

			log.Infof("Worker %d: Processing discovery request: ", workerID)

			responseMap := snmp.Discovery(ip, community, version)

			jsonResponse, _ := json.Marshal(responseMap)

			worker.Send(string(jsonResponse), 0)

		case pollingKey:

			log.Infof("Worker %d: Processing polling request: ", workerID)

			result := snmp.FetchSNMPData(ip, community, version)

			jsonData, _ := json.Marshal(result)

			log.Infof("Worker %d: End time after sending response: %v", workerID)

			worker.Send(string(jsonData), 0)

		default:

			response := map[string]interface{}{

				"status": "fail",

				"message": "Request type not supported currently",
			}

			jsonResponse, _ := json.Marshal(response)

			log.Infof("Worker %d: End time after sending response: %v", workerID)

			worker.Send(string(jsonResponse), 0)
		}
	}
}

// validateRequest checks the presence of required fields in the request data and identifies missing or empty fields.
// It returns a map of valid fields with their corresponding values, and a map of missing fields with error messages.
func validateRequest(reqData map[string]string) (map[string]string, map[string]string) {

	missingFields := make(map[string]string)

	values := make(map[string]string)

	for _, field := range []string{ipKey, communityKey, versionKey, pluginTypeKey, requestTypeKey} {

		value, ok := reqData[field]

		if !ok || value == "" {

			missingFields[field] = errorMissingField

		} else {

			values[field] = value
		}
	}

	return values, missingFields
}
