// auth.go - JWT authentication middleware

package middleware // Declares the package name

import ( // Import required packages
	"go-mqtt-backend/config" // Project config
	"net/http"               // HTTP status codes
	"strings"                // String operations

	"github.com/gin-gonic/gin"     // Gin web framework
	"github.com/golang-jwt/jwt/v5" // JWT library
)

func AuthMiddleware() gin.HandlerFunc { // Returns a Gin middleware function
	return func(c *gin.Context) { // Middleware handler
		header := c.GetHeader("Authorization")                     // Get Authorization header
		if header == "" || !strings.HasPrefix(header, "Bearer ") { // If missing or invalid
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid token"}) // Return 401
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")                               // Remove 'Bearer ' prefix
		cfg := config.Load()                                                            // Load config for JWT secret
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) { // Parse JWT
			return []byte(cfg.JWTSecret), nil // Provide secret key
		})
		if err != nil || !token.Valid { // If invalid
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"}) // Return 401
			return
		}
		c.Next() // Continue to next handler
	}
}
