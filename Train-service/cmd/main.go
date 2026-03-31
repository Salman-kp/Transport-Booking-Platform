package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/nabeel-mp/tripneo/train-service/config"
	"github.com/nabeel-mp/tripneo/train-service/db"
	"github.com/nabeel-mp/tripneo/train-service/jobs"
	"github.com/nabeel-mp/tripneo/train-service/redis"
	"github.com/nabeel-mp/tripneo/train-service/routes"
	"github.com/nabeel-mp/tripneo/train-service/seed"
	"github.com/robfig/cron/v3"
)

func main() {
	cfg := config.LoadConfig()

	db.ConnectPostgres(cfg)

	rdb := redis.Client(cfg.REDIS_HOST, cfg.REDIS_PORT)

	seed.SeedAll(db.DB)

	c := cron.New()
	c.AddFunc("0 2 * * *", func() {
		jobs.GenerateUpcomingInventory(db.DB, 30) // Runs every day at 2 AM
	})
	c.Start()

	// Initial run on boot
	go jobs.GenerateUpcomingInventory(db.DB, 30)

	// kafka.InitProducer(cfg)

	// go grpc.StartServer(cfg)

	app := fiber.New(fiber.Config{
		AppName: "TripNEO Train Service v1.0",
	})

	// 7. Register all routes
	routes.Register(app, cfg, rdb)

	// 8. Start HTTP server
	log.Printf("Train service starting on port %s", cfg.APP_PORT)
	if err := app.Listen(":" + cfg.APP_PORT); err != nil {
		log.Fatal(err)
	}
}
