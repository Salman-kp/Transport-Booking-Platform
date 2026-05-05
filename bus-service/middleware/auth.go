package middleware

import (
	"log"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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

func RequireAdmin() fiber.Handler {
	return func(c fiber.Ctx) error {
		roleStr := c.Get("X-User-Role")
		if roleStr != "admin" && roleStr != "superadmin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: Admin access required",
			})
		}
		return c.Next()
	}
}

func RequireOperator(db *gorm.DB) fiber.Handler {
	return func(c fiber.Ctx) error {
		roleStr := c.Get("X-User-Role")
		// Allowed roles for operator endpoints
		if roleStr != "operator" && roleStr != "superadmin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: Operator access required",
			})
		}

		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized: Missing or invalid userID",
			})
		}

		// Find the OperatorUser
		var opUser model.OperatorUser
		if err := db.Where("user_id = ?", userID).First(&opUser).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Forbidden: You are not associated with any operator",
			})
		}

		// Attach operatorID to context for handlers
		c.Locals("operatorID", opUser.OperatorID)
		return c.Next()
	}
}
