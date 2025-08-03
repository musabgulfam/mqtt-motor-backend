// auth.go - JWT authentication middleware
// This file implements authentication and authorization for the API
//
// Authentication Flow:
// 1. Extract JWT token from Authorization header
// 2. Validate token signature and expiration
// 3. Extract user ID from token claims
// 4. Store user ID in context for handlers
//
// Authorization Flow (Admin):
// 1. Run authentication middleware first
// 2. Extract user ID from context
// 3. Query database for user details
// 4. Check if user has admin role
// 5. Allow/deny access based on role

package middleware // Declares the package name

import ( // Import required packages
	"go-mqtt-backend/config" // Project config (for JWT secret)
	// Database connection (for user queries)
	// User model (for role checking)
	"net/http" // HTTP status codes (401, 403, etc.)
	"strings"  // String operations (for header parsing)

	"github.com/gin-gonic/gin"     // Gin web framework (for middleware)
	"github.com/golang-jwt/jwt/v5" // JWT library (for token validation)
)

// AuthMiddleware - Returns a Gin middleware function for JWT authentication
// This middleware validates JWT tokens and extracts user information
//
// How it works:
// 1. Checks for "Authorization: Bearer <token>" header
// 2. Validates JWT token signature and expiration
// 3. Extracts user ID from token claims
// 4. Stores user ID in Gin context for later use
// 5. Continues to next handler if valid, aborts if invalid
func AuthMiddleware() gin.HandlerFunc { // Returns a Gin middleware function
	return func(c *gin.Context) { // Middleware handler (runs before each request)
		// STEP 1: Extract Authorization header
		// Look for the standard "Bearer token" format in HTTP headers
		header := c.GetHeader("Authorization")                     // Get Authorization header
		if header == "" || !strings.HasPrefix(header, "Bearer ") { // If missing or invalid format
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"}) // Return 401 Unauthorized
			return
		}

		// STEP 2: Parse JWT token
		// Remove "Bearer " prefix and validate the JWT token
		tokenStr := strings.TrimPrefix(header, "Bearer ")                               // Remove 'Bearer ' prefix
		cfg := config.Load()                                                            // Load config for JWT secret
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) { // Parse JWT
			return []byte(cfg.JWTSecret), nil // Provide secret key for validation
		})
		if err != nil || !token.Valid { // If token is invalid or expired
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"}) // Return 401 Unauthorized
			return
		}
		// Example inside your AuthMiddleware
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// JWT numbers are float64 by default
			userIDFloat, ok := claims["sub"].(float64)
			if !ok {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID in token"})
				return
			}
			c.Set("userID", uint(userIDFloat)) // or c.Set("userID", uint(userIDFloat))
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		c.Next() // Continue to next handler
	}
}
