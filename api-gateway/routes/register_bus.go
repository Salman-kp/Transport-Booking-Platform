package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/middleware"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/redis/go-redis/v9"
)

func RegisterBusRoutes(app *fiber.App, cfg *config.Config, rdb *redis.Client) {
	api := app.Group("/api/buses")

	// health
	api.Get("/health", proxy.To(cfg.BUS_SERVICE_URL))

	// ---------------- PUBLIC BUS ROUTES ----------------

	api.Get("/search", proxy.To(cfg.BUS_SERVICE_URL))
	api.Get("/bus-stops", proxy.To(cfg.BUS_SERVICE_URL))
	api.Get("/operators", proxy.To(cfg.BUS_SERVICE_URL))

	api.Get("/:instanceId", proxy.To(cfg.BUS_SERVICE_URL))
	api.Get("/:instanceId/fares", proxy.To(cfg.BUS_SERVICE_URL))
	api.Get("/:instanceId/seats", proxy.To(cfg.BUS_SERVICE_URL))
	api.Get("/:instanceId/amenities", proxy.To(cfg.BUS_SERVICE_URL))
	api.Get("/:instanceId/boarding-points", proxy.To(cfg.BUS_SERVICE_URL))
	api.Get("/:instanceId/dropping-points", proxy.To(cfg.BUS_SERVICE_URL))
	api.Get("/:instanceId/route", proxy.To(cfg.BUS_SERVICE_URL))

	// ---------------- PROTECTED BOOKING ROUTES ----------------

	bookings := api.Group("/bookings",
		middleware.JwtMiddleware(cfg),
		middleware.RateLimit(rdb),
	)

	bookings.Post("/", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.Get("/user/history", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.Get("/pnr/:pnr", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.Get("/:bookingId", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.Post("/:bookingId/confirm", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.Post("/:bookingId/cancel", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.Get("/:bookingId/ticket", proxy.To(cfg.BUS_SERVICE_URL))
}
