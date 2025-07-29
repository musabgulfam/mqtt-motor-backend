// user_test.go - Automated tests for user registration and login handlers
// Run with: go test ./...

package handlers

import (
	"bytes"                    // For building request bodies
	"encoding/json"            // For encoding/decoding JSON
	"go-mqtt-backend/config"   // Project config
	"go-mqtt-backend/database" // Database connection
	"net/http"                 // HTTP status codes
	"net/http/httptest"        // HTTP test helpers
	"os"                       // For file operations
	"testing"                  // Go's testing package

	"github.com/gin-gonic/gin"           // Gin web framework
	"github.com/stretchr/testify/assert" // For assertions
)

// setupTestDB removes any existing test DB and creates a new one for each test run
func setupTestDB() {
	_ = os.Remove("test.db")     // Remove old test DB if exists
	cfg := config.Load()         // Load config
	cfg.DBPath = "test.db"       // Use a separate test DB
	database.Connect(cfg.DBPath) // Connect and migrate
}

// setupRouter returns a Gin engine with the user routes for testing
func setupRouter() *gin.Engine {
	r := gin.Default()            // New Gin router
	r.POST("/register", Register) // Register endpoint
	r.POST("/login", Login)       // Login endpoint
	return r
}

// TestRegisterAndLogin tests user registration and login
func TestRegisterAndLogin(t *testing.T) {
	setupTestDB()           // Prepare test DB
	router := setupRouter() // Prepare router

	// --- Test registration ---
	reg := RegisterInput{
		Email:    "test@example.com",
		Password: "testpass",
	}
	body, _ := json.Marshal(reg)                                          // Encode input as JSON
	w := httptest.NewRecorder()                                           // Record HTTP response
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(body)) // Build request
	req.Header.Set("Content-Type", "application/json")                    // Set header
	router.ServeHTTP(w, req)                                              // Serve request
	assert.Equal(t, 200, w.Code)                                          // Assert success

	// --- Test login ---
	login := LoginInput{
		Email:    "test@example.com",
		Password: "testpass",
	}
	body, _ = json.Marshal(login)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code) // Assert success

	// --- Test login with wrong password ---
	login.Password = "wrongpass"
	body, _ = json.Marshal(login)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, 401, w.Code) // Should be unauthorized
}
