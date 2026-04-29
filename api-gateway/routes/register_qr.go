package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/redis/go-redis/v9"
)

func RegisterQRRoutes(app *gin.Engine, cfg *config.Config, rdb *redis.Client) {
	// Simple public endpoint for fetching dynamic QR codes
	app.GET("/api/qr/generate", proxy.To(cfg.QR_SERVICE_URL))
}
