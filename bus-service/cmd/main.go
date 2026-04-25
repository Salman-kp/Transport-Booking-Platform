package main

import (
	"context"
	"log"

	"github.com/Salman-kp/tripneo/bus-service/config"
	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/handler"
	"github.com/Salman-kp/tripneo/bus-service/jobs"
	"github.com/Salman-kp/tripneo/bus-service/kafka"
	"github.com/Salman-kp/tripneo/bus-service/redis"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/Salman-kp/tripneo/bus-service/routes"
	"github.com/Salman-kp/tripneo/bus-service/rpc"
	"github.com/Salman-kp/tripneo/bus-service/seed"
	"github.com/Salman-kp/tripneo/bus-service/service"
	"github.com/Salman-kp/tripneo/bus-service/ws"
	"github.com/gofiber/fiber/v3"
	"github.com/robfig/cron/v3"
)

func main() {
	cfg := config.LoadConfig()

	db.ConnectPostgres(cfg)
	redis.ConnectRedis(cfg)

	log.Println("RUN_SEED_ON_BOOT =", cfg.RUN_SEED_ON_BOOT)

	if cfg.RUN_SEED_ON_BOOT == "true" {
		seed.SeedAll(db.DB)
	}

	go service.StartRedisExpirySubscriber()

	// initialize gRPC Client
	payClient, err := rpc.NewPaymentClient(cfg.PAYMENT_SERVICE_ADDR)
	if err != nil {
		log.Printf("Warning: Payment gRPC client failed to connect: %v", err)
	} else {
		defer payClient.Close()
	}

	// initialize repos and services for background Workers
	bookingRepo := repository.NewBookingRepository(db.DB)
	bookingService := service.NewBookingService(bookingRepo, redis.Client, payClient, ws.DefaultManager)

	// initialize kafka consumer
	kafkaConsumer := kafka.NewConsumer(cfg.KAFKA_BROKERS, "bus-payment-topic", "bus-service-group")
	if kafkaConsumer != nil {
		defer kafkaConsumer.Close()
		go kafkaConsumer.ConsumePaymentEvents(context.Background(), func(evt kafka.PaymentCompletedEvent) {
			bookingService.ProcessPaymentEvent(evt)
		})
	}

	app := fiber.New()

	app.Get("/api/buses/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	app.Use("/api/buses/ws", handler.WebsocketUpgradeMiddleware)
	app.Get("/api/buses/ws", handler.HandleWebSocket)

	routes.SetupBusRoutes(app, db.DB, cfg)
	routes.SetupBookingRoutes(app, db.DB, payClient, ws.DefaultManager)

	// ── Background jobs ────────────
	c := cron.New()
	c.AddFunc("0 0 * * *", func() {
		jobs.GenerateUpcomingInventory(db.DB)
	})
	c.AddFunc("*/5 * * * *", func() {
		jobs.CleanupExpiredBookings(db.DB)
	})
	c.Start()

	go jobs.GenerateUpcomingInventory(db.DB)

	log.Printf("🚌 Bus Service running on http://localhost:%s", cfg.PORT)
	app.Listen(":" + cfg.PORT)
}
