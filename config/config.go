// config.go - Handles configuration for the project

package config // Declares the package name

import ( // Import required packages
	"os" // For reading environment variables
)

type Config struct { // Config struct holds all configuration values
	DBPath     string // Path to the SQLite database file
	MQTTBroker string // Address of the MQTT broker
	JWTSecret  string // Secret key for JWT authentication
}

func Load() *Config { // Load reads config from environment variables or uses defaults
	return &Config{
		DBPath:     getEnv("DB_PATH", "data.db"),                  // Get DB path or use default
		MQTTBroker: getEnv("MQTT_BROKER", "tcp://localhost:1883"), // Get MQTT broker or use default
		JWTSecret:  getEnv("JWT_SECRET", "supersecret"),           // Get JWT secret or use default
	}
}

func getEnv(key, fallback string) string { // Helper to get env var or fallback
	if value := os.Getenv(key); value != "" { // If env var is set, use it
		return value
	}
	return fallback // Otherwise, use fallback value
}
