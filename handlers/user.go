// user.go - Handles user registration and login

package handlers // Declares the package name

import ( // Import required packages
	"go-mqtt-backend/config"   // Project config
	"go-mqtt-backend/database" // Database connection
	"go-mqtt-backend/models"   // User model
	"net/http"                 // HTTP status codes
	"time"                     // For token expiration

	"github.com/gin-gonic/gin"     // Gin web framework
	"github.com/golang-jwt/jwt/v5" // JWT library
	"golang.org/x/crypto/bcrypt"   // Password hashing
)

type RegisterInput struct { // Struct for registration input
	Email    string `json:"email" binding:"required"`    // Email (required)
	Password string `json:"password" binding:"required"` // Password (required)
}

type LoginInput struct { // Struct for login input
	Email    string `json:"email" binding:"required"`    // Email (required)
	Password string `json:"password" binding:"required"` // Password (required)
}

func Register(c *gin.Context) { // Handler for user registration
	var input RegisterInput                          // Declare input variable
	if err := c.ShouldBindJSON(&input); err != nil { // Parse JSON input
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Return error if invalid
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost) // Hash password
	user := models.User{Email: input.Email, Password: string(hash)}                    // Create user struct
	if err := database.DB.Create(&user).Error; err != nil {                            // Save user to DB
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Return error if DB fails
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "registration successful"}) // Success response
}

func Login(c *gin.Context) { // Handler for user login
	var input LoginInput                             // Declare input variable
	if err := c.ShouldBindJSON(&input); err != nil { // Parse JSON input
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // Return error if invalid
		return
	}
	var user models.User                                                                      // Declare user variable
	if err := database.DB.Where("username = ?", input.Email).First(&user).Error; err != nil { // Find user by email
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"}) // Return error if not found
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil { // Check password
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"}) // Return error if wrong
		return
	}
	// JWT generation
	cfg := config.Load()                                              // Load config for JWT secret
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{ // Create JWT token
		"user_id": user.ID,                               // Add user ID to token
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // Set expiration (72 hours)
	})
	tokenString, _ := token.SignedString([]byte(cfg.JWTSecret)) // Sign token
	c.JSON(http.StatusOK, gin.H{"token": tokenString})          // Return token
}
