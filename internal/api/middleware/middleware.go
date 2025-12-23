package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/afterdarksys/adsops-utils/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Logger logs request details
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("request_id", c.GetString("request_id")),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.String("user_id", userID.(string)))
		}

		if status >= 500 {
			logger.Error("Request completed with error", fields...)
		} else if status >= 400 {
			logger.Warn("Request completed with warning", fields...)
		} else {
			logger.Info("Request completed", fields...)
		}
	}
}

// Recovery handles panics and logs them
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("request_id", c.GetString("request_id")),
					zap.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":       "INTERNAL_ERROR",
						"message":    "An internal error occurred",
						"request_id": c.GetString("request_id"),
						"timestamp":  time.Now().UTC().Format(time.RFC3339),
					},
				})
			}
		}()
		c.Next()
	}
}

// CORS handles Cross-Origin Resource Sharing
func CORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Allow all origins in development, specific origins in production
		if cfg.Environment == "development" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			// In production, check against allowed origins
			allowedOrigins := []string{
				"https://changes.afterdarksys.com",
				"https://api.changes.afterdarksys.com",
			}
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Origin, Cache-Control, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimit implements basic rate limiting
func RateLimit() gin.HandlerFunc {
	// TODO: Implement Redis-based rate limiting
	return func(c *gin.Context) {
		c.Next()
	}
}

// SecurityHeaders adds security-related headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}

// Auth validates JWT tokens and sets user context
func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":      "UNAUTHORIZED",
					"message":   "Authorization header is required",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				},
			})
			return
		}

		// Extract Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":      "INVALID_TOKEN_FORMAT",
					"message":   "Authorization header must be in 'Bearer <token>' format",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				},
			})
			return
		}

		token := parts[1]

		// TODO: Validate JWT token and extract claims
		// For now, we'll just set placeholder values
		_ = token
		c.Set("user_id", "placeholder")
		c.Set("organization_id", "placeholder")
		c.Set("roles", []string{"user"})

		c.Next()
	}
}

// RequireRole checks if the user has one of the required roles
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":      "FORBIDDEN",
					"message":   "Access denied",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				},
			})
			return
		}

		userRoleSlice, ok := userRoles.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":      "INTERNAL_ERROR",
					"message":   "Invalid role configuration",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				},
			})
			return
		}

		for _, required := range roles {
			for _, userRole := range userRoleSlice {
				if userRole == required {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": gin.H{
				"code":      "FORBIDDEN",
				"message":   "You do not have permission to access this resource",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		})
	}
}

// InternalOnly restricts access to internal networks
func InternalOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Allow localhost and private networks
		allowedPrefixes := []string{
			"127.",
			"10.",
			"172.16.", "172.17.", "172.18.", "172.19.",
			"172.20.", "172.21.", "172.22.", "172.23.",
			"172.24.", "172.25.", "172.26.", "172.27.",
			"172.28.", "172.29.", "172.30.", "172.31.",
			"192.168.",
			"::1",
		}

		allowed := false
		for _, prefix := range allowedPrefixes {
			if strings.HasPrefix(clientIP, prefix) {
				allowed = true
				break
			}
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{
					"code":      "FORBIDDEN",
					"message":   "Access denied from external network",
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				},
			})
			return
		}

		c.Next()
	}
}
