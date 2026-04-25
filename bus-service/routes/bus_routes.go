package routes

import (
	"github.com/Salman-kp/tripneo/bus-service/config"
	"github.com/Salman-kp/tripneo/bus-service/handler"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/Salman-kp/tripneo/bus-service/service"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func SetupBusRoutes(app *fiber.App, db *gorm.DB, cfg *config.Config) {

	busRepo := repository.NewBusRepository(db)
	busService := service.NewBusService(busRepo)
	busHandler := handler.NewBusHandler(busService)

	api := app.Group("/api/buses")

	api.Get("/search", busHandler.SearchBuses)
	api.Get("/bus-stops", busHandler.GetBusStops)
	api.Get("/operators", busHandler.GetOperators)

	instance := api.Group("/:instanceId")
	instance.Get("/", busHandler.GetBus)
	instance.Get("/fares", busHandler.GetBusFares)
	instance.Get("/seats", busHandler.GetBusSeats)
	instance.Get("/amenities", busHandler.GetBusAmenities)
	instance.Get("/boarding-points", busHandler.GetBoardingPoints)
	instance.Get("/dropping-points", busHandler.GetDroppingPoints)
	instance.Get("/route", busHandler.GetRoute)
}
