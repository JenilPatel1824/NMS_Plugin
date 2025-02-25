package main

import (
	"GO_Plugin/config"
	"GO_Plugin/handler"
	"GO_Plugin/logger"
)

func main() {

	// Initialize logger
	log := logger.NewLogger()

	// Load configuration
	cfg := config.LoadConfig()

	// Start ZeroMQ Polling Engine
	log.Info("Starting Polling Engine...")
	handler.StartZMQServer(cfg, log)
}
