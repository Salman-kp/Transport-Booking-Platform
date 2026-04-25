package routes

import (
	"github.com/Salman-kp/tripneo/bus-service/handler"
	"github.com/Salman-kp/tripneo/bus-service/middleware"
	busredis "github.com/Salman-kp/tripneo/bus-service/redis"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/Salman-kp/tripneo/bus-service/rpc"
	"github.com/Salman-kp/tripneo/bus-service/service"
	"github.com/Salman-kp/tripneo/bus-service/ws"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func SetupBookingRoutes(app *fiber.App, db *gorm.DB, payClient *rpc.PaymentClient, wsManager *ws.Manager) {

	bookingRepo := repository.NewBookingRepository(db)
	bookingService := service.NewBookingService(bookingRepo, busredis.Client, payClient, wsManager)
	bookingHandler := handler.NewBookingHandler(bookingService)

	api := app.Group("/api/buses/bookings")

	api.Use(middleware.AuthMiddleware)

	api.Post("/", bookingHandler.CreateBooking)
	api.Get("/user/history", bookingHandler.GetUserHistory)
	api.Get("/pnr/:pnr", bookingHandler.GetBookingByPNR)
	api.Get("/:bookingId", bookingHandler.GetBooking)

	api.Post("/:bookingId/confirm", bookingHandler.ConfirmBooking)
	api.Post("/:bookingId/cancel", bookingHandler.CancelBooking)
	api.Get("/:bookingId/ticket", bookingHandler.GetTicket)
}
