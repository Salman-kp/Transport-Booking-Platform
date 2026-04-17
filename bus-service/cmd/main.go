package main

import (
	"log"

	"github.com/Salman-kp/tripneo/bus-service/config"
	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/jobs"
	busredis "github.com/Salman-kp/tripneo/bus-service/redis"
	"github.com/Salman-kp/tripneo/bus-service/redpanda"
	"github.com/Salman-kp/tripneo/bus-service/routes"
	"github.com/Salman-kp/tripneo/bus-service/seed"
	"github.com/gofiber/fiber/v3"
	"github.com/robfig/cron/v3"
)

func main() {
	cfg := config.LoadConfig()

	db.ConnectPostgres(cfg)

	rdb := busredis.Client(cfg.REDIS_HOST, cfg.REDIS_PORT)

	producer := redpanda.NewProducer(cfg.REDPANDA_BROKERS)
	defer producer.Close()

	if cfg.RUN_SEED_ON_BOOT == "true" {
		if err := seed.SeedAll(db.DB); err != nil {
			log.Fatal("Seeding failed:", err)
		}
	}

	// ── Fiber app ────────────────
	app := fiber.New()

	app.Get("/api/bus/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	// Composition root — inject DB + Config + Redis + Redpanda producer
	routes.SetupBusRoutes(app, cfg, db.DB, rdb, producer)

	// ── Background jobs ────────────
	cr := cron.New()
	_, crErr := cr.AddFunc("0 0 * * *", func() {
		jobs.GenerateUpcomingInventory(db.DB)
	})
	if crErr != nil {
		log.Fatal("Failed to register cron job:", crErr)
	}
	cr.Start()
	log.Println("Cron jobs started")

	go jobs.GenerateUpcomingInventory(db.DB)

	log.Printf("🚌 Bus Service running on http://localhost:%s", cfg.PORT)
	if err := app.Listen(":" + cfg.PORT); err != nil {
		log.Fatal(err)
	}
}
