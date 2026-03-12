package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// add unique request id in header
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		uuid := uuid.NewString()

		c.Request().Header.Set("X-Request-ID", uuid)
		return c.Next()
	}
}
