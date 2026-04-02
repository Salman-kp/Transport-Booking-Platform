package handlers

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	domainerrors "github.com/nabeel-mp/tripneo/train-service/domain_errors"
	"github.com/nabeel-mp/tripneo/train-service/kafka"
	"github.com/nabeel-mp/tripneo/train-service/service"
	goredis "github.com/redis/go-redis/v9"
)

var validate = validator.New()

// BookTrain handles POST /api/train/book
func BookTrain(rdb *goredis.Client, producer *kafka.Producer) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, _ := c.Locals("userID").(string)

		var req service.BookingRequest
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}
		if err := validate.Struct(req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}

		resp, err := service.CreateBooking(c.Context(), rdb, userID, req, producer)
		if err != nil {
			statusCode := mapDomainErrorToStatus(err)
			return c.Status(statusCode).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(201).JSON(resp)
	}
}

// GetBooking handles GET /api/train/bookings/:id
func GetBooking(rdb *goredis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, _ := c.Locals("userID").(string)
		bookingID := c.Params("id")

		booking, err := service.GetBooking(bookingID, userID)
		if err != nil {
			return c.Status(mapDomainErrorToStatus(err)).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(200).JSON(booking)
	}
}

// GetBookingHistory handles GET /api/train/bookings/user/history
func GetBookingHistory() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, _ := c.Locals("userID").(string)

		bookings, err := service.GetUserBookingHistory(userID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(200).JSON(fiber.Map{"bookings": bookings})
	}
}

// CancelBooking handles POST /api/train/bookings/:id/cancel
func CancelBooking(rdb *goredis.Client, producer *kafka.Producer) fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, _ := c.Locals("userID").(string)
		bookingID := c.Params("id")

		cancellation, err := service.CancelBookingByUser(c.Context(), rdb, bookingID, userID, producer)
		if err != nil {
			return c.Status(mapDomainErrorToStatus(err)).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(200).JSON(fiber.Map{
			"message":       "booking cancelled",
			"refund_amount": cancellation.RefundAmount,
			"refund_status": cancellation.RefundStatus,
		})
	}
}

// GetBookingPassengers handles GET /api/train/bookings/:id/passengers
func GetBookingPassengers() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, _ := c.Locals("userID").(string)
		bookingID := c.Params("id")

		// Ownership check
		booking, err := service.GetBooking(bookingID, userID)
		if err != nil {
			return c.Status(mapDomainErrorToStatus(err)).JSON(fiber.Map{"error": err.Error()})
		}
		_ = booking

		passengers, err := service.GetPassengers(bookingID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(200).JSON(fiber.Map{"passengers": passengers})
	}
}

// mapDomainErrorToStatus maps domain errors to HTTP status codes.
func mapDomainErrorToStatus(err error) int {
	switch {
	case errors.Is(err, domainerrors.ErrTrainNotFound),
		errors.Is(err, domainerrors.ErrScheduleNotFound),
		errors.Is(err, domainerrors.ErrBookingNotFound),
		errors.Is(err, domainerrors.ErrPNRNotFound):
		return 404
	case errors.Is(err, domainerrors.ErrUnauthorized):
		return 403
	case errors.Is(err, domainerrors.ErrSeatAlreadyLocked):
		return 409
	case errors.Is(err, domainerrors.ErrSeatAlreadyBooked),
		errors.Is(err, domainerrors.ErrNoSeatsAvailable):
		return 409
	case errors.Is(err, domainerrors.ErrBookingNotConfirmed),
		errors.Is(err, domainerrors.ErrCannotCancel):
		return 422
	default:
		return 500
	}
}
