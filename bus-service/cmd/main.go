package main

import (
	"github.com/Salman-kp/tripneo/bus-service/config"
	"github.com/Salman-kp/tripneo/bus-service/db"
	// "github.com/Salman-kp/tripneo/bus-service/routes"

	"log"

	"github.com/gofiber/fiber/v3"
)

func main() {
	// Load config
	cfg := config.LoadConfig()

	// Connect to database
	db.ConnectPostgres(cfg)

	// Redis

	// Service
	// busService := service.NewBusService(rdb)

	// // Handler
	// busHandler := handler.NewBusHandler(busService)

	// Fiber
	app := fiber.New()

	// Routes
	// routes.Register(app, busHandler)

	app.Get("/api/bus/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	if err := app.Listen(":" + cfg.APP_PORT); err != nil {
		log.Fatal(err)
	}
}
