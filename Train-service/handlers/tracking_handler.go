package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nabeel-mp/tripneo/train-service/service"
	goredis "github.com/redis/go-redis/v9"
)

// GetLiveStatus handles GET /api/train/:id/live-status
func GetLiveStatus(rdb *goredis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		scheduleID := c.Params("id")

		result, err := service.GetLiveStatus(c.Context(), rdb, scheduleID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(200).JSON(result)
	}
}
