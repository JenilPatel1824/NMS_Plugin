package config

type Config struct {
	ZMQPort           string
	VertxResponsePort string
	VertxHost         string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		ZMQPort:           "5555",
		VertxResponsePort: "5556",
		VertxHost:         "localhost",
	}
}
