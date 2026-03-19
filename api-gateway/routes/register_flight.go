package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/redis/go-redis/v9"
)

func RegisterFlightRoutes(app *fiber.App, cfg *config.Config, rdb *redis.Client) {
	api := app.Group("/api/flights")

	api.Get("/health", proxy.To(cfg.FLIGHT_SERVICE_URL))

}
