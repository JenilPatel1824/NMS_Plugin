package main

import (
	"GO_Plugin/src/config"
	"GO_Plugin/src/server"
	"GO_Plugin/src/util"
)

// main is the entry point of the application. It initializes logging, loads configuration, and starts the ZeroMQ polling engine.
func main() {

	log := util.NewLogger()

	cfg := config.LoadConfig()

	log.Info("Starting Polling Engine...")

	server.StartPull(cfg, log)
}
