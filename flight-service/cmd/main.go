package main

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/flight-service/config"
	"github.com/junaid9001/tripneo/flight-service/db"
	"github.com/junaid9001/tripneo/flight-service/jobs"
	"github.com/junaid9001/tripneo/flight-service/routes"
	"github.com/junaid9001/tripneo/flight-service/seed"
	"github.com/robfig/cron/v3"
)

func main() {
	cfg := config.LoadConfig()

	db.ConnectPostgres(cfg)

	if cfg.RUN_SEED_ON_BOOT == "true" {
		seed.SeedAll(db.DB)
	}

	app := fiber.New()

	app.Get("/api/flights/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Register all external API Routes
	routes.SetupFlightRoutes(app, db.DB)

	// Start Background Job Scheduler
	c := cron.New()
	c.AddFunc("0 0 * * *", func() {
		jobs.GenerateUpcomingInventory(db.DB)
	})
	c.Start()

	go jobs.GenerateUpcomingInventory(db.DB)

	app.Listen(":" + cfg.APP_PORT)
}

//test
