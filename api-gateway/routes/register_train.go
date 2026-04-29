package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/redis/go-redis/v9"
)

func RegisterTrainRoutes(app *gin.Engine, cfg *config.Config, rdb *redis.Client) {
	api := app.Group("/api/train")

	api.GET("/health", proxy.To(cfg.TRAIN_SERVICE_URL))

	// ------ Public ------
	api.GET("/search", proxy.To(cfg.TRAIN_SERVICE_URL))
	api.GET("/:id", proxy.To(cfg.TRAIN_SERVICE_URL))
	api.GET("/:id/live-status", proxy.To(cfg.TRAIN_SERVICE_URL))
	api.GET("/:id/seats", proxy.To(cfg.TRAIN_SERVICE_URL))

	//----- Protected -----
	api.POST("/book", proxy.To(cfg.TRAIN_SERVICE_URL))
	api.GET("/bookings/:id", proxy.To(cfg.TRAIN_SERVICE_URL))
	api.GET("/bookings/user/history", proxy.To(cfg.TRAIN_SERVICE_URL))
	api.POST("/bookings/:id/cancel", proxy.To(cfg.TRAIN_SERVICE_URL))

}
