package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/middleware"
	"github.com/junaid9001/tripneo/api-gateway/redis"
	"github.com/junaid9001/tripneo/api-gateway/routes"
)

func main() {

	cfg := config.LoadConfig()
	rdb := redis.Client(cfg.REDIS_HOST, cfg.REDIS_PORT)

	app := gin.Default()

	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FRONTEND_URL},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Accept", "Authorization"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	}))

	app.Use(middleware.RequestID())
	app.Use(middleware.PrometheusHTTP())
	app.Use(middleware.IpLimit(rdb))

	routes.Register(app, cfg, rdb)

	if err := app.Run(":" + cfg.APP_PORT); err != nil {
		log.Print(err)
	}
}
