package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/middleware"
	"github.com/junaid9001/tripneo/api-gateway/redis"
	"github.com/junaid9001/tripneo/api-gateway/routes"
)

func main() {

	cfg := config.LoadConfig()
	rdb := redis.Client(cfg.REDIS_HOST, cfg.REDIS_PORT)

	app := fiber.New()
	app.Use(middleware.RequestID())
	app.Use(middleware.IpLimit(rdb))

	routes.Register(app, cfg, rdb)

	if err := app.Listen(":" + cfg.APP_PORT); err != nil {
		log.Print(err)
	}

}
