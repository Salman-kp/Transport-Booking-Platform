package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/middleware"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/redis/go-redis/v9"
)

func RegisterBusRoutes(app *gin.Engine, cfg *config.Config, rdb *redis.Client) {
	api := app.Group("/api/bus")

	// health
	api.GET("/health", proxy.To(cfg.BUS_SERVICE_URL))

	// ---------------- PUBLIC BUS ROUTES ----------------

	api.GET("/search", proxy.To(cfg.BUS_SERVICE_URL))
	api.GET("/bus-stops", proxy.To(cfg.BUS_SERVICE_URL))
	api.GET("/operators", proxy.To(cfg.BUS_SERVICE_URL))

	api.GET("/:instanceId", proxy.To(cfg.BUS_SERVICE_URL))
	api.GET("/:instanceId/fares", proxy.To(cfg.BUS_SERVICE_URL))
	api.GET("/:instanceId/seats", proxy.To(cfg.BUS_SERVICE_URL))
	api.GET("/:instanceId/amenities", proxy.To(cfg.BUS_SERVICE_URL))
	api.GET("/:instanceId/boarding-points", proxy.To(cfg.BUS_SERVICE_URL))
	api.GET("/:instanceId/dropping-points", proxy.To(cfg.BUS_SERVICE_URL))
	api.GET("/:instanceId/route", proxy.To(cfg.BUS_SERVICE_URL))

	// ---------------- PROTECTED BOOKING ROUTES ----------------

	bookings := api.Group("/bookings")
	bookings.Use(middleware.JwtMiddleware(cfg))
	bookings.Use(middleware.RateLimit(rdb))

	bookings.POST("/", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.GET("/user/history", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.GET("/pnr/:pnr", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.GET("/:bookingId", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.POST("/:bookingId/confirm", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.POST("/:bookingId/cancel", proxy.To(cfg.BUS_SERVICE_URL))
	bookings.GET("/:bookingId/ticket", proxy.To(cfg.BUS_SERVICE_URL))
}
