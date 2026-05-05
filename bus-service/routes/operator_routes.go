package routes

import (
	"github.com/Salman-kp/tripneo/bus-service/handler"
	"github.com/Salman-kp/tripneo/bus-service/middleware"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/Salman-kp/tripneo/bus-service/service"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func SetupOperatorRoutes(app *fiber.App, db *gorm.DB) {
	opRepo := repository.NewOperatorRepository(db)
	opService := service.NewOperatorService(opRepo)
	opHandler := handler.NewOperatorHandler(opService)

	opGroup := app.Group("/api/buses/operators")

	// Registration is done by Admin
	opGroup.Post("/register", middleware.RequireAdmin(), opHandler.RegisterOperator)

	// Operator specific routes
	profile := opGroup.Group("", middleware.RequireOperator(db))

	profile.Get("/profile", opHandler.GetProfile)
	profile.Get("/inventory", opHandler.GetInventory)
	profile.Post("/inventory/load", opHandler.LoadInventory)
	profile.Get("/inventory/:id/bookings", opHandler.GetInventoryBookings)
	profile.Get("/analytics", opHandler.GetAnalytics)
}
