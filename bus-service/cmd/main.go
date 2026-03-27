package main

import (
	"log"

	"github.com/Salman-kp/tripneo/bus-service/config"
	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/jobs"
	"github.com/Salman-kp/tripneo/bus-service/routes"
	"github.com/Salman-kp/tripneo/bus-service/seed"
	"github.com/gofiber/fiber/v3"
	"github.com/robfig/cron/v3"
)

func main() {
	cfg := config.LoadConfig()

	db.ConnectPostgres(cfg)

	if cfg.RUN_SEED_ON_BOOT == "true" {
		if err := seed.SeedAll(db.DB); err != nil {
			log.Fatal("Seeding failed:", err)
		}
	}

	app := fiber.New()

	app.Get("/api/bus/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Register all external API Routes
	routes.SetupBusRoutes(app, db.DB)

	// Start Background Job Scheduler
	c := cron.New()
	_, err := c.AddFunc("0 0 * * *", func() {
		jobs.GenerateUpcomingInventory(db.DB)
	})
	if err != nil {
		log.Fatal("Failed to register cron job:", err)
	}
	c.Start()
	log.Println("Cron job started successfully")

	go jobs.GenerateUpcomingInventory(db.DB)

	log.Println("🚀 Bus Service running on http://localhost:" + cfg.PORT)

	if err := app.Listen(":" + cfg.PORT); err != nil {
		log.Fatal(err)
	}
}
