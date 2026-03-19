package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/middleware"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func Register(app *fiber.App, cfg *config.Config, rdb *redis.Client) {

	app.Get("/health", func(c fiber.Ctx) error {
		hdr := c.Get("X-Request-ID")

		return c.Status(200).JSON(fiber.Map{"status": "ok", "request_id": hdr})
	})

	RegisterFlightRoutes(app, cfg, rdb)
	RegisterTrainRoutes(app, cfg, rdb)

	app.Get("/metrics", promhttp.Handler())

	api := app.Group("/api")

	api.Get("/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	auth := api.Group("/auth")

	// public auth routes
	auth.Post("/register", proxy.To(cfg.AUTH_SERVICE_URL))
	auth.Post("/verify-otp", proxy.To(cfg.AUTH_SERVICE_URL))
	auth.Post("/resend-otp", proxy.To(cfg.AUTH_SERVICE_URL))
	auth.Post("/login", proxy.To(cfg.AUTH_SERVICE_URL))

	auth.Post("/logout",
		middleware.JwtMiddleware(cfg),
		middleware.RateLimit(rdb),
		proxy.To(cfg.AUTH_SERVICE_URL),
	)

}
