package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/flight-service/handlers"
	"github.com/junaid9001/tripneo/flight-service/middlewares"
	"github.com/junaid9001/tripneo/flight-service/repository"
	"github.com/junaid9001/tripneo/flight-service/services"
	"gorm.io/gorm"
)

func SetupAdminRoutes(app *fiber.App, db *gorm.DB) {
	bookingRepo := repository.NewBookingRepository(db)
	adminService := services.NewAdminService(bookingRepo, db)
	adminHandler := handlers.NewAdminHandler(adminService)

	admin := app.Group("/api/flights/admin")

	// Global Admin Protection
	admin.Use(middlewares.AdminMiddleware)

	// Booking Management
	admin.Get("/bookings", adminHandler.GetAllBookings)
	admin.Put("/bookings/:id/status", adminHandler.UpdateBookingStatus)

	// Flight Management
	admin.Post("/flights", adminHandler.CreateFlight)
	admin.Put("/flights/:id", adminHandler.UpdateFlight)
	admin.Delete("/flights/:id", adminHandler.DeleteFlight)

	// Pricing Overrides
	admin.Patch("/flights/:instanceId/fares", adminHandler.UpdateFares)
}
