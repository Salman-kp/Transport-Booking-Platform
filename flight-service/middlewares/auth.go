package middlewares

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func RequirePermission(requiredPermission string) fiber.Handler {
	return func(c fiber.Ctx) error {
		roleStr := c.Get("X-User-Role")

		if roleStr == "superadmin" {
			return c.Next()
		}

		permissionsHeader := c.Get("X-User-Permissions")
		permissions := strings.Split(permissionsHeader, ",")

		hasPermission := false
		for _, p := range permissions {
			if strings.TrimSpace(p) == requiredPermission {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: Insufficient permissions",
			})
		}

		return c.Next()
	}
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
