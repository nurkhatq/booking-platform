package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"booking-platform/shared/auth"
	"booking-platform/shared/config"
)

func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

// CORS middleware
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Subdomain")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Logger middleware (alias for RequestLogger)
func Logger() gin.HandlerFunc {
	return RequestLogger()
}

// ErrorHandler middleware
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": err.Error(),
			})
		}
	}
}

// RequestID middleware
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("%d", time.Now().UnixNano())
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// I18n middleware (alias for LanguageDetector)
func I18n(cfg *config.Config) gin.HandlerFunc {
	return LanguageDetector(cfg)
}

func RateLimit(cfg *config.Config) gin.HandlerFunc {
	limiters := make(map[string]*rate.Limiter)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		limiter, exists := limiters[ip]
		if !exists {
			limiter = rate.NewLimiter(
				rate.Every(cfg.Security.RateLimitWindow),
				cfg.Security.RateLimitRequests,
			)
			limiters[ip] = limiter
		}

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func SubdomainExtractor() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.GetHeader("Host")
		subdomain := ""

		if strings.Contains(host, ".") {
			parts := strings.Split(host, ".")
			if len(parts) >= 2 {
				// Extract subdomain (everything before the main domain)
				if parts[len(parts)-2] == "jazyl" && parts[len(parts)-1] == "tech" {
					if len(parts) > 2 {
						subdomain = parts[0]
					}
				}
			}
		}

		// Also check X-Subdomain header (set by Nginx)
		if headerSubdomain := c.GetHeader("X-Subdomain"); headerSubdomain != "" {
			subdomain = headerSubdomain
		}

		c.Set("subdomain", subdomain)
		c.Next()
	}
}

func LanguageDetector(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var language string

		// Check Accept-Language header
		acceptLang := c.GetHeader("Accept-Language")
		if acceptLang != "" {
			// Parse Accept-Language header (simplified)
			langs := strings.Split(acceptLang, ",")
			if len(langs) > 0 {
				lang := strings.TrimSpace(strings.Split(langs[0], ";")[0])
				for _, supported := range cfg.I18n.SupportedLanguages {
					if strings.HasPrefix(lang, supported) {
						language = supported
						break
					}
				}
			}
		}

		// Default to configured default language
		if language == "" {
			language = cfg.I18n.DefaultLanguage
		}

		c.Set("language", language)
		c.Next()
	}
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "missing_auth_token",
				"message": "Authorization header required",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_token",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("user_role", claims.Role)
		c.Set("user_email", claims.Email)

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "role_not_found",
				"message": "User role not found in context",
			})
			c.Abort()
			return
		}

		roleStr := userRole.(string)
		for _, role := range roles {
			if roleStr == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":   "insufficient_permissions",
			"message": "Insufficient permissions for this action",
		})
		c.Abort()
	}
}

func ClientAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "missing_auth_token",
				"message": "Authorization header required",
			})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid_token",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// For client tokens, user_id is actually session_id
		if claims.Role != "CLIENT" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "client_token_required",
				"message": "Client authentication token required",
			})
			c.Abort()
			return
		}

		c.Set("client_session_id", claims.UserID)
		c.Set("client_email", claims.Email)

		c.Next()
	}
}
