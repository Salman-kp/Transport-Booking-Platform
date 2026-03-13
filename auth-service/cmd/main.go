package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/auth-service/config"
	"github.com/junaid9001/tripneo/auth-service/db"
	"github.com/junaid9001/tripneo/auth-service/redis"
	"github.com/junaid9001/tripneo/auth-service/routes"
)

func main() {
	cfg := config.LoadConfig()

	db.ConnectPostgres(cfg)

	rdb := redis.Client(cfg.REDIS_HOST, cfg.REDIS_PORT)

	app := fiber.New()

	routes.Register(app, cfg, rdb)

	if err := app.Listen(":" + cfg.APP_PORT); err != nil {
		log.Print(err)
	}
}
