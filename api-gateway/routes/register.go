package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/middleware"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func Register(app *gin.Engine, cfg *config.Config, rdb *redis.Client) {

	app.GET("/health", func(c *gin.Context) {
		hdr := c.GetHeader("X-Request-ID")

		c.JSON(200, gin.H{"status": "ok", "request_id": hdr})
	})

	RegisterFlightRoutes(app, cfg, rdb)
	RegisterTrainRoutes(app, cfg, rdb)
	RegisterBusRoutes(app, cfg, rdb)
	RegisterChatRoutes(app, cfg, rdb)
	RegisterQRRoutes(app, cfg, rdb)

	app.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := app.Group("/api")

	api.GET("/health", func(c *gin.Context) {
		c.String(200, "ok")
	})

	auth := api.Group("/auth")

	// public auth routes
	auth.POST("/register", proxy.To(cfg.AUTH_SERVICE_URL))
	auth.POST("/verify-otp", proxy.To(cfg.AUTH_SERVICE_URL))
	auth.POST("/resend-otp", proxy.To(cfg.AUTH_SERVICE_URL))
	auth.POST("/login", proxy.To(cfg.AUTH_SERVICE_URL))

	auth.POST("/logout",
		middleware.RateLimit(rdb),
		proxy.To(cfg.AUTH_SERVICE_URL),
	)
}
