package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nabeel-mp/tripneo/train-service/config"
	"github.com/nabeel-mp/tripneo/train-service/handlers"
	"github.com/nabeel-mp/tripneo/train-service/middleware"
	goredis "github.com/redis/go-redis/v9"
)

func Register(app *fiber.App, cfg *config.Config, rdb *goredis.Client) {

	app.Get("/health", func(c fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{
			"status":  "ok",
			"service": "train-service",
		})
	})

	api := app.Group("/api")
	train := api.Group("/train")

	// --- Public routes ---
	train.Get("/search", handlers.SearchTrains(rdb))
	// --- Protected routes ---
	train.Post("/book", middleware.ExtractUser(), handlers.BookTrain(rdb))
	train.Get("/bookings/user/history", middleware.ExtractUser(), handlers.GetBookingHistory())
	train.Get("/bookings/:id", middleware.ExtractUser(), handlers.GetBooking(rdb))
	train.Post("/bookings/:id/cancel", middleware.ExtractUser(), handlers.CancelBooking(rdb))
	train.Get("/tickets/:booking_id", middleware.ExtractUser(), handlers.GetTicket())
	train.Post("/tickets/verify", middleware.ExtractUser(), handlers.VerifyTicket())

	// --- Dynamic :id routes (must come LAST to avoid Fiber param conflicts) ---
	train.Get("/:id/live-status", handlers.GetLiveStatus(rdb))
	train.Get("/:id/seats", handlers.GetSeatMap(rdb))
	train.Get("/:id", handlers.GetTrainByID())
}
