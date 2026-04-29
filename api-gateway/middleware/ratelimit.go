package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ip based ratelimit for all request
func IpLimit(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {

		key := "ratelimit:" + c.ClientIP()

		count, err := rdb.Incr(c.Request.Context(), key).Result()

		if err != nil {
			log.Print(err)
			c.String(500, "internal server error")
			c.Abort()
			return
		}

		if count == 1 {
			rdb.Expire(c.Request.Context(), key, 1*time.Minute)
		}

		if count > 60 {
			c.String(429, "too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}

// userID based ratelimit for private routes
func RateLimit(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr, exists := c.Get("userID")
		if !exists {
			c.String(500, "internal server error")
			c.Abort()
			return
		}
		
		userID, ok := userIDStr.(string)
		if !ok {
			c.String(500, "internal server error")
			c.Abort()
			return
		}
		
		key := "ratelimit:" + userID

		count, err := rdb.Incr(c.Request.Context(), key).Result()

		if err != nil {
			log.Print(err)
			c.String(500, "internal server error")
			c.Abort()
			return
		}

		if count == 1 {
			rdb.Expire(c.Request.Context(), key, 1*time.Minute)
		}

		if count > 500 {
			c.String(429, "too many requests")
			c.Abort()
			return
		}

		c.Next()
	}
}
