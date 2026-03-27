package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nabeel-mp/tripneo/train-service/service"
	goredis "github.com/redis/go-redis/v9"
)

// SearchTrains handles GET /api/train/search
func SearchTrains(rdb *goredis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		from := c.Query("from")
		to := c.Query("to")
		date := c.Query("date")
		class := c.Query("class")

		if from == "" || to == "" || date == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing required search parameters: from, to, and date",
			})
		}

		results, err := service.SearchTrains(c.Context(), rdb, from, to, date, class)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch schedules"})
		}

		// Returning 200 even if empty is often preferred for search results
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"count":  len(results),
			"trains": results,
		})
	}
}

// GetTrainByID handles GET /api/train/:id
func GetTrainByID() fiber.Handler {
	return func(c fiber.Ctx) error {
		scheduleID := c.Params("id")
		schedule, err := service.GetScheduleDetail(scheduleID)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(200).JSON(schedule)
	}
}

// GetSeatMap handles GET /api/train/:id/seats?class=SL
func GetSeatMap(rdb *goredis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		scheduleID := c.Params("id")
		class := c.Query("class", "SL")

		seats, err := service.GetSeatMap(c.Context(), rdb, scheduleID, class)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(200).JSON(fiber.Map{"seats": seats})
	}
}
