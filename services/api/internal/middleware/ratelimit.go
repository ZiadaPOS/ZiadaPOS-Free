package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/troubleman96/ziadapos/internal/cache"
)

// RateLimit is a fixed-window Redis-backed limiter. When Redis is down the
// limiter fails open (allows requests) so the API never hard-crashes.
func RateLimit(cache *cache.Cache, limit int, window time.Duration) gin.HandlerFunc {
	client := cache.Client()
	return func(c *gin.Context) {
		key := rateKey(c)
		ctx := context.Background()

		count, err := client.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}
		if count == 1 {
			client.Expire(ctx, key, window)
		}
		if count > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "too many requests, slow down",
			})
			return
		}
		c.Next()
	}
}

func rateKey(c *gin.Context) string {
	// Prefer an authenticated user id for fairer per-user limits.
	if uid, ok := c.Get("user_id"); ok {
		if s, ok := uid.(string); ok {
			return fmt.Sprintf("rl:user:%s:%s", s, c.FullPath())
		}
	}
	ip := c.ClientIP()
	return fmt.Sprintf("rl:ip:%s:%s", ip, c.FullPath())
}

// compile-time assertion that redis import is used by callers of RateLimit
var _ = redis.Nil
