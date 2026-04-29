package middlewares

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func AdminMiddleware(c fiber.Ctx) error {
	roleStr := c.Get("X-User-Role")

	if roleStr != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Forbidden: Requires admin privileges",
		})
	}

	return c.Next()
}

func AuthMiddleware(c fiber.Ctx) error {
	userIDStr := c.Get("X-User-Id")
	roleStr := c.Get("X-User-Role")

	if userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing X-User-ID header",
		})
	}

	parsedUUID, err := uuid.Parse(userIDStr)
	if err != nil {
		log.Println("Invalid UUID header format:", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid X-User-ID format",
		})
	}

	if roleStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing X-User-Role header",
		})
	}

	c.Locals("userID", parsedUUID)
	c.Locals("role", roleStr)

	return c.Next()
}
