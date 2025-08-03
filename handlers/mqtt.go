// mqtt.go - Handles MQTT commands and motor queue logic

package handlers // Declares the package name

import ( // Import required packages
	"go-mqtt-backend/database"
	"go-mqtt-backend/models"
	"go-mqtt-backend/mqtt" // MQTT client
	"net/http"             // HTTP status codes
	"sync"                 // For mutex (thread safety)
	"time"                 // For time operations

	"github.com/gin-gonic/gin" // Gin web framework
)

type CommandInput struct { // Struct for command input
	Topic   string      `json:"topic" binding:"required"`   // MQTT topic (required)
	Payload interface{} `json:"payload" binding:"required"` // Payload (required)
}

func SendCommand(c *gin.Context) { // Handler to send MQTT command
	var input CommandInput                           // Declare input variable
	if err := c.ShouldBindJSON(&input); err != nil { // Parse JSON input
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Return error if invalid
		return
	}
	if err := mqtt.Publish(input.Topic, input.Payload); err != nil { // Publish to MQTT
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // Return error if publish fails
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "command sent"}) // Success response
}

// For demonstration, this endpoint just returns a placeholder
func GetDeviceData(c *gin.Context) { // Handler to get device data (placeholder)
	c.JSON(http.StatusOK, gin.H{"data": "device data would be here"}) // Return placeholder data
}

type MotorRequest struct { // Struct for motor-on request
	UserID    uint          // User ID (not used in this example)
	RequestAt time.Time     // Time of request
	Duration  time.Duration // How long to turn on
}

var ( // Variables for motor queue and quota
	motorQueue      = make(chan *MotorRequest, 100) // Channel for queued requests
	motorQuotaMutex sync.Mutex                      // Mutex for thread safety
	totalMotorTime  time.Duration                   // Total motor-on time in 24h
	quotaResetTime  time.Time                       // When quota resets
	motorQuota      = 1 * time.Hour                 // Max allowed per 24h
)

func init() { // Initialize quota reset and start queue processor
	quotaResetTime = time.Now().Add(24 * time.Hour) // Set initial reset time
	go processMotorQueue()                          // Start queue processor goroutine
}

func processMotorQueue() { // Goroutine to process motor queue
	for req := range motorQueue { // For each request in queue
		motorQuotaMutex.Lock()                // Lock for thread safety
		if time.Now().After(quotaResetTime) { // If quota period expired
			totalMotorTime = 0                              // Reset total time
			quotaResetTime = time.Now().Add(24 * time.Hour) // Set next reset
		}
		totalMotorTime += req.Duration // Only update total time
		motorQuotaMutex.Unlock()

		// --- Motor control logic ---
		mqtt.Publish("motor/control", "on")  // Send ON command
		time.Sleep(req.Duration)             // Wait for duration
		mqtt.Publish("motor/control", "off") // Send OFF command
	}
}

// Handler to enqueue motor-on requests
func EnqueueMotorRequest(c *gin.Context) {
	var input struct {
		Duration int `json:"duration" binding:"required"` // Duration in minutes
	}
	if err := c.ShouldBindJSON(&input); err != nil { // Parse JSON input
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Return error if invalid
		return
	}
	motorQuotaMutex.Lock()                // Lock for thread safety
	if time.Now().After(quotaResetTime) { // If quota period expired
		totalMotorTime = 0                              // Reset total time
		quotaResetTime = time.Now().Add(24 * time.Hour) // Set next reset
	}
	if totalMotorTime+time.Duration(input.Duration)*time.Minute > motorQuota { // If quota exceeded
		motorQuotaMutex.Unlock()                                                                                      // Unlock
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Daily motor-on quota reached. Try again after 24 hours."}) // Return error
		return
	}
	motorQuotaMutex.Unlock()          // Unlock
	userID, exists := c.Get("userID") // Get user ID from context
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in token"})
		return
	}
	// Log to DB
	logEntry := models.DeviceActivation{
		UserID:    userID.(uint),
		RequestAt: time.Now(),
		Duration:  time.Duration(input.Duration) * time.Minute,
	}
	if err := database.DB.Create(&logEntry).Error; err != nil {
		// Optionally handle/log DB error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to log request"})
		return
	}
	// In a real app, get user ID from JWT claims
	motorQueue <- &MotorRequest{ // Add request to queue
		UserID:    0,
		RequestAt: time.Now(),
		Duration:  time.Duration(input.Duration) * time.Minute,
	}
	c.JSON(http.StatusOK, gin.H{"message": "Request queued"}) // Success response
}
