package handler

import (
	"github.com/Salman-kp/tripneo/bus-service/dto"
	"github.com/Salman-kp/tripneo/bus-service/pkg/utils"
	"github.com/Salman-kp/tripneo/bus-service/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type BookingHandler struct {
	svc service.BookingService
}

func NewBookingHandler(svc service.BookingService) *BookingHandler {
	return &BookingHandler{svc: svc}
}

func (h *BookingHandler) getAuthUserID(c fiber.Ctx) string {
	if val := c.Locals("userID"); val != nil {
		if uid, ok := val.(uuid.UUID); ok {
			return uid.String()
		}
	}
	return ""
}

func (h *BookingHandler) CreateBooking(c fiber.Ctx) error {
	userID := h.getAuthUserID(c)
	if userID == "" {
		return utils.Fail(c, fiber.StatusUnauthorized, "Unauthorized user")
	}

	var req dto.CreateBookingRequest
	if err := c.Bind().JSON(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	booking, err := h.svc.CreateBooking(userID, req)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.Success(c, fiber.StatusCreated, "Booking created successfully", booking)
}

func (h *BookingHandler) GetBooking(c fiber.Ctx) error {
	userID := h.getAuthUserID(c)
	if userID == "" {
		return utils.Fail(c, fiber.StatusUnauthorized, "Unauthorized user")
	}

	bookingID := c.Params("bookingId")
	booking, err := h.svc.GetBookingByID(bookingID, userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "Booking not found")
	}

	return utils.Success(c, fiber.StatusOK, "Booking retrieved successfully", booking)
}

func (h *BookingHandler) GetBookingByPNR(c fiber.Ctx) error {
	userID := h.getAuthUserID(c)
	if userID == "" {
		return utils.Fail(c, fiber.StatusUnauthorized, "Unauthorized user")
	}

	pnr := c.Params("pnr")
	booking, err := h.svc.GetBookingByPNR(pnr, userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "Booking not found")
	}

	return utils.Success(c, fiber.StatusOK, "Booking retrieved successfully", booking)
}

func (h *BookingHandler) GetUserHistory(c fiber.Ctx) error {
	userID := h.getAuthUserID(c)
	if userID == "" {
		return utils.Fail(c, fiber.StatusUnauthorized, "Unauthorized user")
	}

	bookings, err := h.svc.GetUserBookings(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "History retrieved successfully", bookings)
}

func (h *BookingHandler) ConfirmBooking(c fiber.Ctx) error {
	userID := h.getAuthUserID(c)
	if userID == "" {
		return utils.Fail(c, fiber.StatusUnauthorized, "Unauthorized user")
	}

	bookingID := c.Params("bookingId")
	secret, err := h.svc.InitiatePayment(bookingID, userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Payment initiated successfully", fiber.Map{
		"stripe_client_secret": secret,
	})
}

func (h *BookingHandler) CancelBooking(c fiber.Ctx) error {
	userID := h.getAuthUserID(c)
	if userID == "" {
		return utils.Fail(c, fiber.StatusUnauthorized, "Unauthorized user")
	}

	bookingID := c.Params("bookingId")

	var req dto.CancelBookingRequest
	_ = c.Bind().JSON(&req)

	if result, err := h.svc.CancelBooking(bookingID, userID, &req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, err.Error())
	} else {
		return utils.Success(c, fiber.StatusOK, "Booking cancelled successfully", result)
	}
}

func (h *BookingHandler) GetTicket(c fiber.Ctx) error {
	userID := h.getAuthUserID(c)
	if userID == "" {
		return utils.Fail(c, fiber.StatusUnauthorized, "Unauthorized user")
	}

	bookingID := c.Params("bookingId")
	ticket, err := h.svc.GetBookingTicket(bookingID, userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "Ticket not found")
	}

	return utils.Success(c, fiber.StatusOK, "Ticket retrieved successfully", ticket)
}
