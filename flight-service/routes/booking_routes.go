package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/flight-service/config"
	"github.com/junaid9001/tripneo/flight-service/handlers"
	"github.com/junaid9001/tripneo/flight-service/middlewares"
	"github.com/junaid9001/tripneo/flight-service/repository"
	"github.com/junaid9001/tripneo/flight-service/rpc"
	"github.com/junaid9001/tripneo/flight-service/services"
	"github.com/junaid9001/tripneo/flight-service/ws"
	"gorm.io/gorm"
)

func SetupBookingRoutes(app *fiber.App, db *gorm.DB, payClient *rpc.PaymentClient, wsManager *ws.Manager, cfg *config.Config) {
	bookingRepo := repository.NewBookingRepository(db)
	bookingService := services.NewBookingService(bookingRepo, payClient, wsManager, cfg.QR_PUBLIC_BASE_URL, cfg.QR_SIGNING_SECRET)
	bookingHandler := handlers.NewBookingHandler(bookingService)

	api := app.Group("/api/flights/bookings")

	api.Use(middlewares.AuthMiddleware)

	api.Post("", bookingHandler.CreateBooking)
	api.Get("/user/history", bookingHandler.GetUserHistory)
	api.Get("/pnr/:pnr", bookingHandler.GetBookingByPNR)
	api.Get("/:bookingId", bookingHandler.GetBookingByID)

	api.Post("/:bookingId/confirm", bookingHandler.ConfirmBooking)
	api.Post("/:bookingId/cancel", bookingHandler.CancelBooking)
	api.Get("/:bookingId/ticket", bookingHandler.GetTicket)
}
