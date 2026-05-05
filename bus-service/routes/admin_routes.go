package routes

import (
	"github.com/Salman-kp/tripneo/bus-service/handler"
	"github.com/Salman-kp/tripneo/bus-service/middleware"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/Salman-kp/tripneo/bus-service/service"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func SetupAdminRoutes(app *fiber.App, db *gorm.DB) {
	adminRepo := repository.NewAdminRepository(db)
	adminService := service.NewAdminService(adminRepo, db)
	adminHandler := handler.NewAdminHandler(adminService)

	admin := app.Group("/api/buses/admin")

	// Global Admin Protection
	admin.Use(middleware.RequireAdmin())

	// Bus Management
	admin.Post("/buses", adminHandler.CreateBus)
	admin.Put("/buses/:id", adminHandler.UpdateBus)
	admin.Get("/buses", adminHandler.ListBuses)
	
	// Bus Instance Management (Cron handles generation, Admin manages specific trips)
	admin.Delete("/instances/:id", adminHandler.DeleteBusInstance)
	admin.Put("/instances/:id/status", adminHandler.UpdateBusInstanceStatus)

	// Operator Management
	admin.Put("/operators/:id/approve", adminHandler.ApproveOperator)
	admin.Put("/operators/:id/suspend", adminHandler.SuspendOperator)

	// Booking & Analytics
	admin.Get("/bookings", adminHandler.GetAllBookings)
	admin.Get("/analytics/revenue", adminHandler.GetRevenueAnalytics)
	admin.Get("/analytics/operators", adminHandler.GetOperatorAnalytics)
	admin.Get("/analytics/upcoming", adminHandler.GetUpcomingTrips)

	// Pricing Rules
	admin.Get("/pricing-rules", adminHandler.GetPricingRules)
	admin.Post("/pricing-rules", adminHandler.CreatePricingRule)
	admin.Put("/pricing-rules/:id", adminHandler.UpdatePricingRule)

	// Cancellation Policies
	admin.Get("/cancellation-policies", adminHandler.GetCancellationPolicies)
	admin.Post("/cancellation-policies", adminHandler.CreateCancellationPolicy)
	admin.Put("/cancellation-policies/:id", adminHandler.UpdateCancellationPolicy)

	// Bus Dependencies Management
	admin.Post("/bus-types", adminHandler.CreateBusType)
	admin.Get("/bus-types", adminHandler.GetBusTypes)
	admin.Post("/bus-stops", adminHandler.CreateBusStop)
	admin.Get("/bus-stops", adminHandler.GetBusStops)
	admin.Post("/operators", adminHandler.CreateOperator)
	admin.Get("/operators", adminHandler.GetOperators)
}
