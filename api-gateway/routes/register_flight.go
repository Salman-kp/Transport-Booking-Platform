package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/middleware"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/redis/go-redis/v9"
)

func RegisterFlightRoutes(app *gin.Engine, cfg *config.Config, rdb *redis.Client) {
	api := app.Group("/api/flights")

	api.GET("/health", proxy.To(cfg.FLIGHT_SERVICE_URL))

	// secure websocket explicitly
	api.GET("/ws", middleware.JwtMiddleware(cfg), proxy.To(cfg.FLIGHT_SERVICE_URL))

	// public flight routes
	api.GET("/search", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.GET("/status/:pnr", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.GET("/:instanceId", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.GET("/:instanceId/fares", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.GET("/:instanceId/seats", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.GET("/:instanceId/ancillaries", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.GET("/:instanceId/fare-prediction", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.GET("/airports", proxy.To(cfg.FLIGHT_SERVICE_URL))
	api.GET("/airlines", proxy.To(cfg.FLIGHT_SERVICE_URL))

	// protected booking routes
	bookings := api.Group("/bookings",
		middleware.JwtMiddleware(cfg),
		middleware.RateLimit(rdb),
	)

	bookings.POST("", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.GET("/user/history", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.GET("/pnr/:pnr", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.GET("/:bookingId", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.POST("/:bookingId/confirm", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.POST("/:bookingId/cancel", proxy.To(cfg.FLIGHT_SERVICE_URL))

	bookings.GET("/:bookingId/ticket", proxy.To(cfg.FLIGHT_SERVICE_URL))

	// admin routes
	admin := api.Group("/admin",
		middleware.JwtMiddleware(cfg),
	)
	admin.Any("/*any", proxy.To(cfg.FLIGHT_SERVICE_URL))
}
