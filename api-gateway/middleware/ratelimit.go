package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
)

// ip based ratelimit for all request
func IpLimit(rdb *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {

		key := "ratelimit:" + c.IP()

		count, err := rdb.Incr(c.RequestCtx(), key).Result()

		if err != nil {
			log.Print(err)
			return c.Status(500).SendString("internal server error")
		}

		if count == 1 {
			rdb.Expire(c.RequestCtx(), key, 1*time.Minute)
		}

		if count > 60 {
			return c.Status(429).SendString("too many requests")
		}

		return c.Next()
	}
}

// userID based ratelimit for private routes
func RateLimit(rdb *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		userIDStr := c.Locals("userID")
		userID, ok := userIDStr.(string)
		if !ok {
			return c.Status(500).SendString("internal server error")
		}
		key := "ratelimit:" + userID

		count, err := rdb.Incr(c.RequestCtx(), key).Result()

		if err != nil {
			log.Print(err)
			return c.Status(500).SendString("internal server error")
		}

		if count == 1 {
			rdb.Expire(c.RequestCtx(), key, 1*time.Minute)
		}

		if count > 500 {
			return c.Status(429).SendString("too many requests")
		}

		return c.Next()
	}
}
