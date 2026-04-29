package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/junaid9001/tripneo/api-gateway/config"
	"github.com/junaid9001/tripneo/api-gateway/middleware"
	"github.com/junaid9001/tripneo/api-gateway/proxy"
	"github.com/redis/go-redis/v9"
)

func RegisterChatRoutes(app *gin.Engine, cfg *config.Config, rdb *redis.Client) {
	api := app.Group("/api")

	chat := api.Group("/chat")

	// WS route BEFORE rate limiter — matching flight-service pattern
	chat.GET("/ws", middleware.JwtMiddleware(cfg), proxy.To(cfg.CHAT_SERVICE_URL))

	// Rate limit REST routes only
	chat.GET("/messages", middleware.JwtMiddleware(cfg), middleware.RateLimit(rdb), proxy.To(cfg.CHAT_SERVICE_URL))
	chat.POST("/admin/reply/:userId", middleware.JwtMiddleware(cfg), middleware.RateLimit(rdb), proxy.To(cfg.CHAT_SERVICE_URL))
}
