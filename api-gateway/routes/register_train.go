package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/redis/go-redis/v9"
)

func RegisterTrainRoutes(app *fiber.App, cfg *config.Config, rdb *redis.Client) {
	api := app.Group("/api/train")

	api.Get("/health", proxy.To(cfg.TRAIN_SERVICE_URL))

}
