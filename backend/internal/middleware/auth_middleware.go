package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"clawreef/internal/models"
	"clawreef/internal/repository"
	"clawreef/internal/runtimeauth"
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

		claims, err := validateUserAccessToken(tokenString)
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

// RuntimeCapabilityVerifier verifies a capability against the current assignment.
// Implementations must honor context cancellation and deadlines.
type RuntimeCapabilityVerifier interface {
	Verify(context.Context, string, string) (runtimeauth.Identity, error)
}

// GatewayAuthWithRuntimeCapabilities adds model capability authentication and
// rejects legacy instance credentials for protected assignments.
func GatewayAuthWithRuntimeCapabilities(instanceRepo repository.InstanceRepository, verifier RuntimeCapabilityVerifier, protectedInstance func(context.Context, int) (bool, error), bindingRepos ...repository.InstanceRuntimeBindingRepository) gin.HandlerFunc {
	return gatewayAuth(instanceRepo, verifier, protectedInstance, true, bindingRepos...)
}

// GatewayAuth accepts either a normal user access JWT or an instance lifecycle gateway token.
func GatewayAuth(instanceRepo repository.InstanceRepository, bindingRepos ...repository.InstanceRuntimeBindingRepository) gin.HandlerFunc {
	return gatewayAuth(instanceRepo, nil, nil, false, bindingRepos...)
}

func gatewayAuth(instanceRepo repository.InstanceRepository, verifier RuntimeCapabilityVerifier, protectedInstance func(context.Context, int) (bool, error), capabilities bool, bindingRepos ...repository.InstanceRuntimeBindingRepository) gin.HandlerFunc {
	var bindingRepo repository.InstanceRuntimeBindingRepository
	if len(bindingRepos) > 0 {
		bindingRepo = bindingRepos[0]
	}
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

		var instance *models.Instance
		var verified runtimeauth.Identity
		isCapability := capabilities && strings.HasPrefix(tokenString, "v1.")
		if isCapability {
			if verifier == nil || protectedInstance == nil || instanceRepo == nil {
				rejectGatewayToken(c)
				return
			}
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			identity, err := verifier.Verify(ctx, tokenString, runtimeauth.AudienceModel)
			cancel()
			if err != nil || identity.InstanceID <= 0 || identity.UserID <= 0 || identity.Generation <= 0 || identity.PodUID == "" {
				rejectGatewayToken(c)
				return
			}
			instance, err = instanceRepo.GetByID(identity.InstanceID)
			if err != nil || instance == nil || instance.ID != identity.InstanceID || instance.UserID != identity.UserID || instance.RuntimeGeneration != identity.Generation || (instance.Status != "running" && instance.Status != "creating") || gatewayInstanceMode(instance.InstanceMode, instance.RuntimeType) != "lite" {
				rejectGatewayToken(c)
				return
			}
			ctx, cancel = context.WithTimeout(c.Request.Context(), 2*time.Second)
			protected, err := protectedInstance(ctx, instance.ID)
			cancel()
			if err != nil || !protected {
				rejectGatewayToken(c)
				return
			}
			verified = identity
		} else {
			if claims, err := validateUserAccessToken(tokenString); err == nil {
				c.Set("userID", claims.UserID)
				c.Set("gatewayAuthType", "user")
				c.Next()
				return
			}
			if instanceRepo == nil {
				rejectGatewayToken(c)
				return
			}
			var err error
			instance, err = instanceRepo.GetByAccessToken(tokenString)
			if err != nil || instance == nil || instance.AccessToken == nil || strings.TrimSpace(*instance.AccessToken) == "" {
				rejectGatewayToken(c)
				return
			}
			if capabilities {
				if protectedInstance == nil {
					rejectGatewayToken(c)
					return
				}
				ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
				protected, err := protectedInstance(ctx, instance.ID)
				cancel()
				if err != nil || protected {
					rejectGatewayToken(c)
					return
				}
			}
		}

		c.Set("userID", instance.UserID)
		c.Set("instanceID", instance.ID)
		c.Set("instanceType", strings.TrimSpace(instance.Type))
		c.Set("instanceMode", gatewayInstanceMode(instance.InstanceMode, instance.RuntimeType))
		c.Set("runtimeType", strings.TrimSpace(instance.RuntimeType))
		if bindingRepo != nil {
			binding, err := bindingRepo.GetRunningByInstanceID(c.Request.Context(), instance.ID)
			if isCapability && (err != nil || (binding != nil && (binding.InstanceID != verified.InstanceID || binding.Generation != verified.Generation || binding.State != "running"))) {
				rejectGatewayToken(c)
				return
			}
			if err == nil && binding != nil {
				c.Set("gatewayID", strings.TrimSpace(binding.GatewayID))
				c.Set("runtimePodID", binding.RuntimePodID)
			}
		}
		c.Set("gatewayAuthType", "instance")
		if isCapability {
			serveCapabilityModel(c, verifier, tokenString, verified)
			return
		}
		c.Next()
	}
}

func rejectGatewayToken(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid gateway token"})
}

func gatewayInstanceMode(instanceMode, runtimeType string) string {
	mode := strings.ToLower(strings.TrimSpace(instanceMode))
	if mode == "lite" || mode == "pro" {
		return mode
	}
	switch strings.ToLower(strings.TrimSpace(runtimeType)) {
	case "gateway":
		return "lite"
	case "desktop", "shell":
		return "pro"
	default:
		return mode
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
	if token := strings.TrimSpace(c.GetHeader("x-api-key")); token != "" {
		return token, true
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

func validateUserAccessToken(tokenString string) (*utils.TokenClaims, error) {
	jwtSecret := getJWTSecret()
	claims, err := utils.ValidateToken(tokenString, jwtSecret)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != "access" {
		return nil, errors.New("invalid token type")
	}
	return claims, nil
}
