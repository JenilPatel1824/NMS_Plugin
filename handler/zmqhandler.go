package handler

import (
	"GO_Plugin/config"
	"GO_Plugin/snmp"
	"encoding/json"
	"fmt"
	"github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"
	"time"
)

// StartZMQServer initializes and runs a ZeroMQ REP server for handling SNMP data requests and sending JSON responses.
func StartZMQServer(cfg *config.Config, log *logrus.Logger) {
	// Create a ZeroMQ REP socket
	socket, err := zmq4.NewSocket(zmq4.REP)

	if err != nil {
		log.Errorf("Failed to create socket: %v", err)
		return
	}
	defer socket.Close()

	bindAddr := fmt.Sprintf("tcp://*:%s", cfg.ZMQPort)

	err = socket.Bind(bindAddr)
	if err != nil {
		log.Errorf("Failed to bind socket: %v", err)
		return
	}
	log.Infof("Polling engine listening on %s...", bindAddr)

	for {
		// Wait for a request
		req, err := socket.Recv(0)

		startTime := time.Now() // Start time tracking
		if err != nil {
			log.Errorf("Failed to receive message: %v", err)
			continue
		}
		log.Infof("Received request: %s", req)

		// Parse the incoming JSON request
		var reqData map[string]string
		if err := json.Unmarshal([]byte(req), &reqData); err != nil {
			log.Errorf("Failed to unmarshal request: %v", err)
			continue
		}

		ip := reqData["ip"]

		community := reqData["community"]

		version := reqData["version"]

		// Fetch SNMP data using the extracted values
		data := snmp.FetchSNMPData(ip, community, version)

		// Convert the data to JSON
		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Errorf("Failed to marshal JSON: %v", err)
			continue
		}

		// Send the JSON response back
		_, err = socket.Send(string(jsonData), 0)
		if err != nil {
			log.Errorf("Failed to send response: %v", err)
		} else {
			log.Infof("Sent response: %s", string(jsonData))
		}

		// Calculate and log the time taken for the request
		timeTaken := time.Since(startTime)

		log.Infof("Time taken for request: %v", timeTaken)
	}
}
