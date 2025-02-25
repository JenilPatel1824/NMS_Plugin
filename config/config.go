package config

type Config struct {
	ZMQPort string
}

// LoadConfig initializes and returns a Config object with default values for the application settings.
func LoadConfig() *Config {
	// Hardcoded for local use
	return &Config{
		ZMQPort: "5555", // Local Port
	}
}
