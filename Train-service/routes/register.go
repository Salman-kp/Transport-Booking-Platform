package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nabeel-mp/tripneo/train-service/config"
	"github.com/nabeel-mp/tripneo/train-service/handlers"
	"github.com/nabeel-mp/tripneo/train-service/middleware"
	"github.com/redis/go-redis/v9"
)

func Register(app *fiber.App, cfg *config.Config, rdb *redis.Client) {

	// Health check — useful for Docker + load balancer probes
	app.Get("/health", func(c fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{"status": "ok", "service": "train-service"})
	})

	api := app.Group("/api")
	train := api.Group("/train")

	train.Get("/health", func(c fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{"status": "ok", "service": "train-service"})
	})

	// --- Public Routes (no auth needed) ---
	train.Get("/search", handlers.SearchTrains())
	train.Post("/tickets/verify", handlers.VerifyTicket())

	// --- Protected Routes (require X-User-ID from gateway) ---
	train.Get("/bookings/:id", middleware.ExtractUser(), handlers.GetBooking())
	train.Post("/bookings/:id/cancel", middleware.ExtractUser(), handlers.CancelBooking())
	train.Get("/tickets/:booking_id", middleware.ExtractUser(), handlers.GetTicket())
	train.Post("/book", middleware.ExtractUser(), handlers.BookTrain())

	// --- Train detail + tracking (public) ---
	// Note: these must come AFTER /search and /tickets to avoid Fiber param conflicts
	train.Get("/:id/live-status", handlers.GetLiveStatus())
	train.Get("/:id", handlers.GetTrainByID())
}
