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
	"go-mqtt-backend/database" // Database connection (for user queries)
	"go-mqtt-backend/models" // User model (for role checking)
	"net/http"               // HTTP status codes (401, 403, etc.)
	"strings"                // String operations (for header parsing)

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
		
		// STEP 3: Extract user ID from token and store in context for later use
		// This allows subsequent handlers to access the user ID without re-parsing the token
		// JWT claims contain the user information that was stored when the token was created
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userID, exists := claims["user_id"]; exists {
				c.Set("user_id", userID) // Store user ID in Gin context
			}
		}
		
		c.Next() // Continue to next handler (authentication successful)
	}
}

// AdminMiddleware - Returns a Gin middleware function for admin access control
// This middleware extends AuthMiddleware to check if the user has admin role
// It implements role-based access control (RBAC) for admin endpoints
// 
// How it works:
// 1. Runs AuthMiddleware first (ensures user is authenticated)
// 2. Extracts user ID from context (set by AuthMiddleware)
// 3. Queries database for user details
// 4. Checks if user has "admin" role
// 5. Allows access only if user is admin
func AdminMiddleware() gin.HandlerFunc { // Returns a Gin middleware function for admin access
	return func(c *gin.Context) { // Middleware handler (runs before admin endpoints)
		// STEP 1: Run the standard authentication middleware first
		// This ensures the user is authenticated before checking their role
		// If authentication fails, this middleware will abort the request
		AuthMiddleware()(c) // Call the auth middleware on the same context
		
		// STEP 2: Check if authentication failed
		// If the auth middleware aborted the request, don't continue
		// This prevents unnecessary database queries for unauthenticated requests
		if c.IsAborted() {
			return // Exit early - authentication failed
		}
		
		// STEP 3: Extract user ID from context (set by AuthMiddleware)
		// The user ID was stored in the context by the AuthMiddleware
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in token"})
			return
		}
		
		// STEP 4: Convert user ID to uint (JWT numbers are stored as float64)
		// JWT stores all numbers as float64, but our database uses uint
		// This type conversion is necessary for database queries
		userID, ok := userIDInterface.(float64) // JWT numbers are float64
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid user ID format"})
			return
		}
		
		// STEP 5: Query database to get user details and check role
		// This implements the role-based access control
		// We need to query the database because the JWT token doesn't contain role information
		var user models.User
		if err := database.DB.First(&user, uint(userID)).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}
		
		// STEP 6: Check if user has admin role
		// Only users with role="admin" can access admin endpoints
		// This is the core of role-based access control (RBAC)
		if user.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		
		c.Next() // Continue to next handler (admin access granted)
	}
}
