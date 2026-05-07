package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/flight-service/config"
	"github.com/junaid9001/tripneo/flight-service/handlers"
	"github.com/junaid9001/tripneo/flight-service/repository"
	"github.com/junaid9001/tripneo/flight-service/services"
	"gorm.io/gorm"
)

func SetupFlightRoutes(app *fiber.App, db *gorm.DB, cfg *config.Config) {

	flightRepo := repository.NewFlightRepository(db)
	flightService := services.NewFlightService(flightRepo)
	flightHandler := handlers.NewFlightHandler(flightService)

	api := app.Group("/api/flights")

	bookingRepo := repository.NewBookingRepository(db)
	trackingService := services.NewTrackingService(bookingRepo, cfg)
	trackingHandler := handlers.NewTrackingHandler(trackingService)

	api.Get("/status/:pnr", trackingHandler.GetFlightStatus)
	api.Get("/search", flightHandler.Search)

	api.Get("/:instanceId/fares", flightHandler.GetFares)
	api.Get("/:instanceId/seats", flightHandler.GetSeatMap)
	api.Get("/:instanceId/ancillaries", flightHandler.GetAncillaries)
	api.Get("/:instanceId/fare-prediction", flightHandler.GetFarePrediction)
	api.Get("/airports", flightHandler.SearchAirport)
	api.Get("/airlines", flightHandler.GetAirlines)
	api.Get("/aircraft-types", flightHandler.GetAircraftTypes)
	api.Get("/:instanceId", flightHandler.GetFlightDetails)
}
