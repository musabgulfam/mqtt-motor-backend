// mqtt.go - Handles MQTT commands and motor queue logic
// This file implements the core motor control system with:
// 1. FIFO queue for motor requests
// 2. Daily quota enforcement (1 hour per 24h)
// 3. Admin force shutdown capability
// 4. Thread-safe operations with mutexes

package handlers // Declares the package name

import ( // Import required packages
	"go-mqtt-backend/mqtt" // MQTT client for sending commands to ESP32
	"net/http"             // HTTP status codes (200, 401, 503, etc.)
	"sync"                 // For mutex (thread safety in concurrent operations)
	"time"                 // For time operations (duration, timestamps)

	"github.com/gin-gonic/gin" // Gin web framework for HTTP handlers
	"go-mqtt-backend/database" // Database client for user queries
	"go-mqtt-backend/models"   // User model for role checking
)

// CommandInput - Structure for MQTT command requests
// Used by the /api/send endpoint to send commands to ESP32
type CommandInput struct { // Struct for command input
	Topic   string      `json:"topic" binding:"required"`   // MQTT topic (required) - e.g., "motor/control"
	Payload interface{} `json:"payload" binding:"required"` // Payload (required) - e.g., "on", "off", or JSON
}

// SendCommand - HTTP handler to send MQTT commands to ESP32
// This is a general-purpose endpoint for sending any MQTT command
func SendCommand(c *gin.Context) { // Handler to send MQTT command
	var input CommandInput                           // Declare input variable
	if err := c.ShouldBindJSON(&input); err != nil { // Parse JSON input from request body
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Return 400 error if JSON is invalid
		return
	}
	if err := mqtt.Publish(input.Topic, input.Payload); err != nil { // Publish to MQTT broker
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}) // Return 500 error if MQTT fails
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "command sent"}) // Return 200 success response
}

// GetDeviceData - Placeholder handler for device data
// In a real application, this would fetch sensor data from ESP32
func GetDeviceData(c *gin.Context) { // Handler to get device data (placeholder)
	c.JSON(http.StatusOK, gin.H{"data": "device data would be here"}) // Return placeholder data
}

// MotorRequest - Structure representing a motor-on request
// Each request contains user info, timing, and duration
type MotorRequest struct { // Struct for motor-on request
	UserID    uint          // User ID (not used in this example, but available for audit)
	RequestAt time.Time     // Time of request (for tracking and audit)
	Duration  time.Duration // How long to turn on (e.g., 30 seconds, 2 minutes)
}

// Global variables for motor queue and quota management
// These are shared across all HTTP requests and the background processor
var ( // Variables for motor queue and quota
	// Queue Management
	motorQueue      = make(chan *MotorRequest, 100) // Channel for queued requests (buffered, max 100)
	motorQuotaMutex sync.Mutex                      // Mutex for thread safety (prevents race conditions)
	
	// Quota Management (24-hour rolling window)
	totalMotorTime  time.Duration                   // Total motor-on time in current 24h period
	quotaResetTime  time.Time                       // When quota resets (next 24h boundary)
	motorQuota      = 1 * time.Hour                 // Max allowed per 24h (configurable)
	
	// Admin shutdown control - Global state for emergency shutdown
	// These variables control the emergency shutdown system
	shutdownMutex   sync.Mutex                      // Mutex for shutdown state (thread safety)
	isShutdown      bool                            // Whether system is in shutdown mode (true = shutdown, false = normal)
	shutdownReason  string                          // Reason for shutdown (e.g., "Emergency maintenance")
	shutdownBy      string                          // Who initiated shutdown (admin email for audit)
	shutdownAt      time.Time                       // When shutdown was initiated (timestamp for audit)
)

// init - Package initialization function (runs once when package is imported)
// Sets up the initial quota reset time and starts the background queue processor
func init() { // Initialize quota reset and start queue processor
	quotaResetTime = time.Now().Add(24 * time.Hour) // Set initial reset time (24h from now)
	go processMotorQueue()                          // Start queue processor goroutine (runs in background)
}

// processMotorQueue - Background goroutine that processes motor requests sequentially
// This function runs continuously and handles the core queue algorithm with admin shutdown integration
// 
// Algorithm Flow:
// 1. Check if system is in admin shutdown mode
// 2. If shutdown: skip processing (drop request)
// 3. If normal: check quota limits
// 4. If quota OK: execute motor control
// 5. If quota exceeded: skip processing (drop request)
func processMotorQueue() { // Goroutine to process motor queue
	for req := range motorQueue { // For each request in queue (FIFO processing - First In, First Out)
		// STEP 1: Check if system is in admin shutdown mode
		// This is the first check before any quota or processing logic
		// Admin shutdown takes priority over all other operations
		shutdownMutex.Lock()                // Lock for thread safety (prevent concurrent access)
		if isShutdown {                     // If system is in shutdown mode
			shutdownMutex.Unlock()          // Unlock immediately (don't hold lock longer than needed)
			// Skip processing during shutdown - request is effectively dropped
			// This prevents any motor operations during emergency shutdown
			continue                        // Move to next request in queue
		}
		shutdownMutex.Unlock()              // Unlock if not in shutdown mode
		
		// STEP 2: Quota management (original queue algorithm)
		// This implements the daily quota system (1 hour per 24h)
		motorQuotaMutex.Lock()              // Lock for thread safety (prevent concurrent quota updates)
		if time.Now().After(quotaResetTime) { // If quota period expired (24h reset)
			totalMotorTime = 0                              // Reset total time to 0
			quotaResetTime = time.Now().Add(24 * time.Hour) // Set next reset time
		}
		if totalMotorTime+req.Duration > motorQuota { // If quota exceeded (1 hour limit)
			motorQuotaMutex.Unlock()        // Unlock
			// Quota exceeded, skip this request (request is dropped)
			continue                        // Move to next request in queue
		}
		totalMotorTime += req.Duration      // Add to total time (quota tracking)
		motorQuotaMutex.Unlock()           // Unlock quota management
		
		// STEP 3: Motor control execution (currently commented out for safety)
		// In production, this would:
		// 1. Send "on" command to motor via MQTT
		// 2. Wait for the specified duration
		// 3. Send "off" command to motor via MQTT
		// --- Motor control logic (commented out) ---
		// mqtt.Publish("motor/control", "on") // Send ON command
		// time.Sleep(req.Duration)             // Wait for duration
		// mqtt.Publish("motor/control", "off") // Send OFF command
		// ------------------------------------------
	}
}

// EnqueueMotorRequest - Handler to enqueue motor-on requests with admin shutdown check
// This function implements the enhanced queue algorithm with admin shutdown integration
// 
// Request Flow:
// 1. Check if system is in admin shutdown mode
// 2. If shutdown: return 503 Service Unavailable
// 3. If normal: validate quota limits
// 4. If quota OK: enqueue request
// 5. If quota exceeded: return 429 Too Many Requests
func EnqueueMotorRequest(c *gin.Context) {
	// STEP 1: Admin shutdown check (NEW - prevents new requests during shutdown)
	// This is the first validation before any quota checking
	// Admin shutdown takes absolute priority over all motor operations
	shutdownMutex.Lock()                   // Lock for thread safety
	if isShutdown {                        // If system is in shutdown mode
		shutdownMutex.Unlock()             // Unlock immediately
		// Return 503 Service Unavailable with shutdown details
		// This provides clear feedback about why the request was rejected
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Motor system is currently shut down",
			"reason": shutdownReason,       // Why it was shut down
			"shutdown_by": shutdownBy,      // Who shut it down
			"shutdown_at": shutdownAt,      // When it was shut down
		})
		return                             // Exit early - no further processing
	}
	shutdownMutex.Unlock()                 // Unlock if not in shutdown mode
	
	// STEP 2: Parse request input (original logic)
	// Extract duration from JSON request body
	var input struct {
		Duration int `json:"duration" binding:"required"` // Duration in seconds
	}
	if err := c.ShouldBindJSON(&input); err != nil { // Parse JSON input from request body
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Return 400 error if JSON is invalid
		return
	}
	
	// STEP 3: Quota validation (original queue algorithm)
	// Check if this request would exceed the daily quota limit
	motorQuotaMutex.Lock()                 // Lock for thread safety
	if time.Now().After(quotaResetTime) {  // If quota period expired (24h reset)
		totalMotorTime = 0                              // Reset total time to 0
		quotaResetTime = time.Now().Add(24 * time.Hour) // Set next reset time
	}
	if totalMotorTime+time.Duration(input.Duration)*time.Second > motorQuota { // If quota exceeded
		motorQuotaMutex.Unlock()           // Unlock
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Daily motor-on quota reached. Try again after 24 hours."}) // Return 429 error
		return
	}
	motorQuotaMutex.Unlock()               // Unlock quota management
	
	// STEP 4: Enqueue request (original logic)
	// Add the request to the processing queue
	// In a real app, get user ID from JWT claims for audit trail
	motorQueue <- &MotorRequest{           // Add request to queue (non-blocking)
		UserID:    0,                      // Placeholder user ID (would be extracted from JWT)
		RequestAt: time.Now(),             // Current timestamp
		Duration:  time.Duration(input.Duration) * time.Second, // Convert to duration
	}
	c.JSON(http.StatusOK, gin.H{"message": "Request queued"}) // Return 200 success response
}

// AdminForceShutdown - Admin handler to force shutdown the motor system
// This function implements the emergency shutdown capability
// 
// Emergency Shutdown Process:
// 1. Validate shutdown reason (required)
// 2. Extract admin user info from JWT token
// 3. Set global shutdown state (thread-safe)
// 4. Send immediate "off" command to motor
// 5. Return success with audit trail
func AdminForceShutdown(c *gin.Context) {
	// STEP 1: Parse shutdown reason from request
	// The reason is required for audit trail and accountability
	var input struct {
		Reason string `json:"reason" binding:"required"` // Reason for shutdown (required)
	}
	if err := c.ShouldBindJSON(&input); err != nil { // Parse JSON input from request body
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Return 400 error if JSON is invalid
		return
	}
	
	// STEP 2: Extract admin user info from JWT token context
	// This gets the user ID that was stored by the middleware
	userIDInterface, exists := c.Get("user_id") // Get user ID from middleware
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in token"})
		return
	}
	
	// Convert user ID to uint (JWT numbers are stored as float64)
	// This is because JWT stores all numbers as float64, but our DB uses uint
	userID, ok := userIDInterface.(float64) // JWT numbers are float64
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID format"})
		return
	}
	
	// STEP 3: Get user details from database for audit trail
	// This ensures we have the complete user info for the audit log
	var user models.User
	if err := database.DB.First(&user, uint(userID)).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	
	// STEP 4: Set shutdown state (thread-safe)
	// This is the critical section that changes the global shutdown state
	shutdownMutex.Lock()                   // Lock for thread safety
	isShutdown = true                      // Set shutdown flag (prevents new requests)
	shutdownReason = input.Reason          // Store shutdown reason (for audit)
	shutdownBy = user.Email                // Store admin email (for audit)
	shutdownAt = time.Now()                // Store shutdown timestamp (for audit)
	shutdownMutex.Unlock()                 // Unlock immediately
	
	// STEP 5: Send immediate shutdown command to motor (safety measure)
	// This ensures the motor stops immediately, even if there are requests in queue
	// This is a critical safety feature for emergency situations
	mqtt.Publish("motor/control", "off")  // Send immediate OFF command
	
	// STEP 6: Return success response with shutdown details
	// This provides complete audit trail information
	c.JSON(http.StatusOK, gin.H{
		"message": "Motor system shut down successfully",
		"reason": shutdownReason,           // Why it was shut down
		"shutdown_by": shutdownBy,          // Who shut it down
		"shutdown_at": shutdownAt,          // When it was shut down
	})
}

// AdminRestart - Admin handler to restart the motor system
// This function clears the shutdown state and resumes normal operation
// 
// Restart Process:
// 1. Clear all shutdown state variables (thread-safe)
// 2. Resume normal queue processing
// 3. Return success confirmation
func AdminRestart(c *gin.Context) {
	// STEP 1: Clear shutdown state (thread-safe)
	// This is the critical section that resumes normal operation
	shutdownMutex.Lock()                   // Lock for thread safety
	isShutdown = false                     // Clear shutdown flag (allow new requests)
	shutdownReason = ""                    // Clear shutdown reason
	shutdownBy = ""                        // Clear admin email
	shutdownAt = time.Time{}               // Clear shutdown timestamp
	shutdownMutex.Unlock()                 // Unlock immediately
	
	// STEP 2: Return success response
	// Confirm that the system has been restarted
	c.JSON(http.StatusOK, gin.H{
		"message": "Motor system restarted successfully",
		"restart_at": time.Now(),          // When it was restarted
	})
}

// GetSystemStatus - Handler to get comprehensive system status
// This function provides visibility into both quota and shutdown states
// 
// Status Information:
// - Shutdown state (active/inactive)
// - Quota usage (current/total/reset time)
// - Queue length (pending requests)
// - Audit trail (who/when/why shutdown occurred)
func GetSystemStatus(c *gin.Context) {
	// STEP 1: Get shutdown status (thread-safe)
	// This provides the current emergency shutdown state
	shutdownMutex.Lock()                   // Lock for thread safety
	shutdownStatus := gin.H{
		"is_shutdown": isShutdown,         // Current shutdown state (true/false)
		"reason": shutdownReason,           // Why it was shut down (if applicable)
		"shutdown_by": shutdownBy,          // Who shut it down (if applicable)
		"shutdown_at": shutdownAt,          // When it was shut down (if applicable)
	}
	shutdownMutex.Unlock()                 // Unlock immediately
	
	// STEP 2: Get quota status (thread-safe)
	// This provides the current quota usage information
	motorQuotaMutex.Lock()                 // Lock for thread safety
	quotaStatus := gin.H{
		"total_motor_time": totalMotorTime.String(), // Current total motor time
		"quota_limit": motorQuota.String(),          // Daily quota limit (1 hour)
		"quota_reset_time": quotaResetTime,          // When quota resets
		"queue_length": len(motorQueue),             // Number of requests in queue
	}
	motorQuotaMutex.Unlock()               // Unlock immediately
	
	// STEP 3: Return combined status
	// This provides a complete picture of system state
	c.JSON(http.StatusOK, gin.H{
		"shutdown_status": shutdownStatus,  // Admin shutdown information
		"quota_status": quotaStatus,        // Quota and queue information
	})
}
