package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/flight-service/handlers"
	"github.com/junaid9001/tripneo/flight-service/repository"
	"github.com/junaid9001/tripneo/flight-service/services"
	"gorm.io/gorm"
)

func SetupFlightRoutes(app *fiber.App, db *gorm.DB) {

	flightRepo := repository.NewFlightRepository(db)
	flightService := services.NewFlightService(flightRepo)
	flightHandler := handlers.NewFlightHandler(flightService)

	api := app.Group("/api/flights")

	api.Get("/search", flightHandler.Search)
	api.Get("/:instanceId", flightHandler.GetFlightDetails)
	api.Get("/:instanceId/fares", flightHandler.GetFares)
	api.Get("/:instanceId/seats", flightHandler.GetSeatMap)
	api.Get("/:instanceId/ancillaries", flightHandler.GetAncillaries)
	api.Get("/:instanceId/fare-prediction", flightHandler.GetFarePrediction)
}
