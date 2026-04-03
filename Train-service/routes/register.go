package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nabeel-mp/tripneo/train-service/config"
	"github.com/nabeel-mp/tripneo/train-service/handlers"
	"github.com/nabeel-mp/tripneo/train-service/kafka"
	"github.com/nabeel-mp/tripneo/train-service/middleware"
	goredis "github.com/redis/go-redis/v9"
)

func Register(app *fiber.App, cfg *config.Config, rdb *goredis.Client, producer *kafka.Producer) {

	app.Get("/health", func(c fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{
			"status":  "ok",
			"service": "train-service",
		})
	})

	api := app.Group("/api")
	train := api.Group("/train")

	// ── Public routes ────────────────────────────────────────────────
	train.Get("/search", handlers.SearchTrains(rdb))
	train.Get("/stations", handlers.GetStations())
	train.Post("/tickets/verify", handlers.VerifyTicket())

	// ── Protected — Bookings ─────────────────────────────────────────
	auth := middleware.ExtractUser()
	train.Post("/book", auth, handlers.BookTrain(rdb, producer))
	train.Get("/bookings/user/history", auth, handlers.GetBookingHistory())
	train.Get("/bookings/:id", auth, handlers.GetBooking(rdb))
	train.Get("/bookings/:id/passengers", auth, handlers.GetBookingPassengers())
	train.Post("/bookings/:id/cancel", auth, handlers.CancelBooking(rdb, producer))
	train.Get("/tickets/:booking_id", auth, handlers.GetTicket())

	// ── Internal (called by Payment Service — no user auth) ──────────
	internal := train.Group("/internal")
	internal.Post("/payment/callback", handlers.PaymentCallback(rdb, producer))

	// ── Admin routes ─────────────────────────────────────────────────
	admin := train.Group("/admin", auth, middleware.RequireRole("admin"))
	admin.Post("/trains", handlers.AdminCreateTrain())
	admin.Put("/trains/:id", handlers.AdminUpdateTrain())
	admin.Post("/schedules/generate", handlers.AdminGenerateSchedules())
	admin.Post("/inventory/purchase", handlers.AdminPurchaseInventory())
	admin.Get("/bookings", handlers.AdminGetBookings())
	admin.Get("/analytics/revenue", handlers.AdminGetRevenue())
	admin.Put("/pricing-rules/:id", handlers.AdminUpdatePricingRule())

	// ── Dynamic :id routes (MUST come last to avoid param conflicts) ──
	train.Get("/:id/classes", handlers.GetClasses())
	train.Get("/:id/live-status", handlers.GetLiveStatus(rdb))
	train.Get("/:id/seats", handlers.GetSeatMap(rdb))
	train.Post("/:scheduleId/seats/:seatId/lock", auth, handlers.LockSeat(rdb))
	train.Delete("/:scheduleId/seats/:seatId/lock", auth, handlers.UnlockSeat(rdb))
	train.Get("/:id", handlers.GetTrainByID())
}
