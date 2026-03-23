package middleware

import (
	"net/http"
	"os"
	"strings"

	"clawreef/internal/utils"
	"github.com/gin-gonic/gin"
)

// Auth middleware validates JWT token
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := extractToken(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authorization header required",
			})
			c.Abort()
			return
		}

		// Get JWT secret from config (in production, should be from env)
		jwtSecret := getJWTSecret()

		// Validate token
		claims, err := utils.ValidateToken(tokenString, jwtSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Check token type
		if claims.TokenType != "access" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid token type",
			})
			c.Abort()
			return
		}

		// Set user ID in context
		c.Set("userID", claims.UserID)
		c.Next()
	}
}

func extractToken(c *gin.Context) (string, bool) {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return "", false
		}
		return parts[1], true
	}

	// Browsers cannot set custom Authorization headers for native WebSocket
	// handshakes, so allow `?token=` specifically for upgrade requests.
	if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
		token := strings.TrimSpace(c.Query("token"))
		if token != "" {
			return token, true
		}
	}

	return "", false
}

func getJWTSecret() string {
	// Get from environment variable, fallback to default
	// Must match the secret used in config.go
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	return "clawreef-dev-secret-key-change-in-production"
}
