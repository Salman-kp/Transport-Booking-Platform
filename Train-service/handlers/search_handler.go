package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nabeel-mp/tripneo/train-service/service"
	goredis "github.com/redis/go-redis/v9"
)

// SearchTrains handles GET /api/train/search
// Query params: from, to, class, date (YYYY-MM-DD)
func SearchTrains(rdb *goredis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		origin := c.Query("from")
		destination := c.Query("to")
		class := c.Query("class", "SL")
		date := c.Query("date")

		if origin == "" || destination == "" || date == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "from, to, and date are required",
			})
		}

		results, err := service.SearchTrains(c.Context(), rdb, origin, destination, class, date)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(200).JSON(fiber.Map{
			"results": results,
			"count":   len(results),
		})
	}
}

// GetTrainByID handles GET /api/train/:id
// :id is a schedule UUID
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
