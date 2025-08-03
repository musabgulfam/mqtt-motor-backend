// main.go - Entry point for the Go MQTT backend server

package main // Declares the package name

import ( // Import required packages
	"go-mqtt-backend/config"     // Project config management
	"go-mqtt-backend/database"   // Database connection and setup
	"go-mqtt-backend/handlers"   // HTTP handlers for API endpoints
	"go-mqtt-backend/middleware" // Middleware (e.g., authentication)
	"go-mqtt-backend/mqtt"       // MQTT client logic
	"log"                        // Logging

	"github.com/gin-gonic/gin" // Gin web framework
)

func main() { // Main function, program entry point
	// STEP 1: Load configuration and establish connections
	cfg := config.Load() // Load configuration (DB path, MQTT broker, JWT secret)

	if err := database.Connect(cfg.DBPath); err != nil { // Connect to the database
		log.Fatal("DB connection error: ", err) // If error, log and exit
	}
	if err := mqtt.Connect(cfg.MQTTBroker); err != nil { // Connect to the MQTT broker
		log.Fatal("MQTT connection error: ", err) // If error, log and exit
	}

	// STEP 2: Create Gin router and configure routes
	r := gin.Default() // Create a new Gin router (web server)

	// Public routes (no authentication required)
	r.POST("/register", handlers.Register) // Public route: user registration
	r.POST("/login", handlers.Login)       // Public route: user login

	// Protected routes (require JWT authentication)
	// These endpoints require a valid JWT token but no specific role
	api := r.Group("/api")               // Create a route group for protected endpoints
	api.Use(middleware.AuthMiddleware()) // Apply JWT authentication middleware
	{
		api.POST("/send", handlers.SendCommand)          // Protected: send MQTT command
		api.GET("/device", handlers.GetDeviceData)       // Protected: get device data
		api.POST("/motor", handlers.EnqueueMotorRequest) // Protected: enqueue motor request
	}

	// STEP 3: Start the web server
	r.Run(":8080") // Start the web server on port 8080
}
