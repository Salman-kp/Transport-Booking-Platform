package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/nabeel-mp/tripneo/train-service/config"
	"github.com/nabeel-mp/tripneo/train-service/db"
	"github.com/nabeel-mp/tripneo/train-service/jobs"
	"github.com/nabeel-mp/tripneo/train-service/kafka"
	"github.com/nabeel-mp/tripneo/train-service/redis"
	"github.com/nabeel-mp/tripneo/train-service/routes"
	"github.com/nabeel-mp/tripneo/train-service/seed"
	"github.com/nabeel-mp/tripneo/train-service/service"
	"github.com/robfig/cron/v3"
)

func main() {
	// 1. Load configuration
	cfg := config.LoadConfig()

	// 2. Connect PostgreSQL + AutoMigrate + seed defaults
	db.ConnectPostgres(cfg)

	// 3. Connect Redis (non-fatal if unavailable in local dev)
	rdb := redis.Client(cfg.REDIS_HOST, cfg.REDIS_PORT)

	// 4. Seed stations + trains from JSON files
	if err := seed.SeedAll(db.DB); err != nil {
		log.Printf("[main] Seed warning: %v", err)
	}

	// 5. Generate schedules + inventory for the next 30 days (sync first run)
	if err := jobs.GenerateUpcomingInventory(db.DB, 30); err != nil {
		log.Printf("[main] Inventory generation warning: %v", err)
	}

	// 6. Schedule daily inventory generation at 2 AM
	c := cron.New()
	c.AddFunc("0 2 * * *", func() {
		jobs.GenerateUpcomingInventory(db.DB, 30)
	})
	c.Start()

	// 7. Initialize Kafka producer (nil if no broker configured — safe)
	producer := kafka.NewProducer(cfg.KAFKA_BROKERS)
	defer producer.Close()

	// 8. Start Kafka consumer (no-op if broker not configured)
	go kafka.StartConsumer(cfg, db.DB, rdb, producer)

	// 9. Start background workers
	go service.RunExpiryWorker(db.DB, rdb, producer)
	go service.RunPricingEngine(db.DB, cfg)

	// 10. Start Fiber HTTP server
	app := fiber.New(fiber.Config{
		AppName: "TripNEO Train Service v1.0",
	})

	routes.Register(app, cfg, rdb, producer)

	log.Printf("[main] Train service starting on port %s", cfg.APP_PORT)
	if err := app.Listen(":" + cfg.APP_PORT); err != nil {
		log.Fatal(err)
	}
}
