package config

type Config struct {
	ZMQPort string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		ZMQPort: "5555",
	}
}
