// Package middleware provides HTTP middleware for the Gin router.
package middleware

import (
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestID assigns a UUID to every request and echoes it back in the
// X-Request-Id header for end-to-end tracing.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.GetHeader("X-Request-Id"))
		if _, err := uuid.Parse(id); err != nil {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Writer.Header().Set("X-Request-Id", id)
		c.Next()
	}
}

// Logger logs each request with method, path, status, duration and request id.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		rid, _ := c.Get("request_id")
		log.Printf("http %s %s -> %d (%s) rid=%v",
			c.Request.Method, c.Request.URL.Path, c.Writer.Status(), duration.Round(time.Millisecond), rid)
	}
}

// Recovery recovers from panics and returns a JSON 500 instead of crashing.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				// Panic payloads can contain request or credential data. Keep logs
				// non-sensitive rather than serializing arbitrary panic values.
				log.Printf("panic: recovered unexpected server error")
				c.AbortWithStatusJSON(500, gin.H{
					"success": false,
					"message": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}
