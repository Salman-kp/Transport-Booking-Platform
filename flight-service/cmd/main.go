package main

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/flight-service/config"
	"github.com/junaid9001/tripneo/flight-service/db"
)

func main() {
	cfg := config.LoadConfig()

	db.ConnectPostgres(cfg)

	app := fiber.New()

	app.Get("/api/flights/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	app.Listen(":" + cfg.APP_PORT)
}
