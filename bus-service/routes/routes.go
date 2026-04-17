package routes

import (
	"github.com/Salman-kp/tripneo/bus-service/config"
	"github.com/Salman-kp/tripneo/bus-service/handler"
	"github.com/Salman-kp/tripneo/bus-service/middleware"
	"github.com/Salman-kp/tripneo/bus-service/redpanda"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/Salman-kp/tripneo/bus-service/service"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func SetupBusRoutes(app *fiber.App, cfg *config.Config, db *gorm.DB, rdb *redis.Client, producer *redpanda.Producer) {

	busRepo := repository.NewBusRepository(db)
	bookingRepo := repository.NewBookingRepository(db)

	busService := service.NewBusService(busRepo)
	bookingService := service.NewBookingService(bookingRepo, rdb, producer)

	// Inject the consumer binding immediately against the repository
	redpanda.StartConsumer(cfg, db, rdb, producer, bookingRepo)

	busHandler := handler.NewBusHandler(busService)
	bookingHandler := handler.NewBookingHandler(bookingService)

	api := app.Group("/api/buses")

	//----------------------- PUBLIC ENDPOINTS -----------------------

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

	//----------------------- PROTECTED ENDPOINTS -----------------------

	bookings := api.Group("/bookings")
	bookings.Use(middleware.AuthMiddleware)

	bookings.Post("/", bookingHandler.CreateBooking)
	bookings.Get("/user/history", bookingHandler.GetUserHistory)
	bookings.Get("/pnr/:pnr", bookingHandler.GetBookingByPNR)
	bookings.Get("/:bookingId", bookingHandler.GetBooking)
	bookings.Post("/:bookingId/confirm", bookingHandler.ConfirmBooking)
	bookings.Post("/:bookingId/cancel", bookingHandler.CancelBooking)
	bookings.Get("/:bookingId/ticket", bookingHandler.GetTicket)
}
