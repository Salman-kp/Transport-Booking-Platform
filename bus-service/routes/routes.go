package routes

import (
	"github.com/Salman-kp/tripneo/bus-service/handler"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/Salman-kp/tripneo/bus-service/service"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func SetupBusRoutes(app *fiber.App, db *gorm.DB) {

	busRepo := repository.NewBusRepository(db)
	busService := service.NewBusService(busRepo)
	busHandler := handler.NewBusHandler(busService)

	api := app.Group("/api/buses")

	//----------------------- PUBLIC ENDPOINTS -----------------------

	api.Get("/search", busHandler.SearchBuses)

	api.Get("/:instanceId/fares", busHandler.GetBusFares)
	api.Get("/:instanceId/seats", busHandler.GetBusSeats)
	api.Get("/:instanceId/amenities", busHandler.GetBusAmenities)
	api.Get("/:instanceId/boarding-points", busHandler.GetBoardingPoints)

	api.Get("/:instanceId", busHandler.GetBus)

	//----------------------- PRIVATE ENDPOINTS  -----------------------

}
