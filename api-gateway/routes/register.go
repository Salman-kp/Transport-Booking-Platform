package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/middleware"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/redis/go-redis/v9"
)

func Register(app *fiber.App, cfg *config.Config, rdb *redis.Client) {

	app.Get("/health", func(c fiber.Ctx) error {
		hdr := c.Get("X-Request-ID")

		return c.Status(200).JSON(fiber.Map{"status": "ok", "request_id": hdr})
	})

	public := app.Group("/api")
	private := app.Group("/ap")

	private.Use(middleware.JwtMiddleware(cfg))
	private.Use(middleware.RateLimit(rdb))

	public.Get("/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	private.Get("/protected_health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	app.All("/auth/*", proxy.To(cfg.AUTH_SERVICE_URL))

}
