package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/auth-service/config"
	"github.com/junaid9001/tripneo/auth-service/handlers"
	"github.com/redis/go-redis/v9"
)

func Register(app *fiber.App, cfg *config.Config, rdb *redis.Client) {
	auth := app.Group("/api/auth")

	auth.Post("/register", handlers.Register(rdb, cfg))
	auth.Post("/verify-otp", handlers.VerifyOtp(rdb))
	auth.Post("/resend-otp", handlers.ResendOtp(cfg, rdb))
	auth.Post("/login", handlers.Login(cfg))
	auth.Post("/logout", handlers.Logout())

	// Admin
	auth.Post("/admin/assign-role", handlers.AssignRole())
	auth.Get("/admin/users", handlers.ListUsers())

	// Internal service-to-service
	auth.Get("/internal/user/:id", handlers.GetUserByID())
}
