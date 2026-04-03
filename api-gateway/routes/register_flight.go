package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/middleware"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/redis/go-redis/v9"
)

func RegisterFlightRoutes(app *fiber.App, cfg *config.Config, rdb *redis.Client) {
	api := app.Group("/api/flights")

	api.Get("/health", proxy.To(cfg.FLIGHT_SERVICE_URL))

	// public flight routes
	api.Get("/search", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.Get("/:instanceId", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.Get("/:instanceId/fares", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.Get("/:instanceId/seats", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.Get("/:instanceId/ancillaries", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.Get("/:instanceId/fare-prediction", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.Get("/airports", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.Get("/airlines", proxy.To(cfg.FLIGHT_SERVICE_URL))

	// protected booking routes
	bookings := api.Group("/bookings",
		middleware.JwtMiddleware(cfg),
		middleware.RateLimit(rdb),
	)

	bookings.Post("/", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.Get("/user/history", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.Get("/pnr/:pnr", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.Get("/:bookingId", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.Post("/:bookingId/confirm", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.Post("/:bookingId/cancel", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.Get("/:bookingId/ticket", proxy.To(cfg.FLIGHT_SERVICE_URL))

}
