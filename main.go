package main

import (
	"GO_Plugin/config"
	"GO_Plugin/handler"
	"GO_Plugin/logger"
)

// main is the entry point of the application. It initializes logging, loads configuration, and starts the ZeroMQ polling engine.
func main() {

	log := logger.NewLogger()

	cfg := config.LoadConfig()

	log.Info("Starting Polling Engine...")

	handler.StartZMQRouter(cfg, log)
}
