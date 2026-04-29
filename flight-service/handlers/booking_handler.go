package handlers

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/junaid9001/tripneo/flight-service/dto"
	"github.com/junaid9001/tripneo/flight-service/services"
)

type BookingHandler struct {
	service *services.BookingService
}

func NewBookingHandler(service *services.BookingService) *BookingHandler {
	return &BookingHandler{service}
}

func (h *BookingHandler) CreateBooking(c fiber.Ctx) error {
	userIDVal := c.Locals("userID")
	userIDStr := ""
	if uid, ok := userIDVal.(uuid.UUID); ok {
		userIDStr = uid.String()
	}

	var req dto.CreateBookingRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	resp, err := h.service.CreateBooking(userIDStr, &req)
	if err != nil {
		log.Printf("CreateBooking Error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *BookingHandler) GetBookingByID(c fiber.Ctx) error {
	id := c.Params("bookingId")
	booking, err := h.service.GetBookingByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "booking not found"})
	}

	return c.JSON(booking)
}

func (h *BookingHandler) GetBookingByPNR(c fiber.Ctx) error {
	pnr := c.Params("pnr")
	booking, err := h.service.GetBookingByPNR(pnr)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "booking not found"})
	}

	return c.JSON(booking)
}

func (h *BookingHandler) GetUserHistory(c fiber.Ctx) error {
	userIDVal := c.Locals("userID")
	userIDStr := ""
	if uid, ok := userIDVal.(uuid.UUID); ok {
		userIDStr = uid.String()
	}

	bookings, err := h.service.GetBookingsByUserID(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(bookings)
}

func (h *BookingHandler) ConfirmBooking(c fiber.Ctx) error {
	id := c.Params("bookingId")
	userIDVal := c.Locals("userID")
	userIDStr := ""
	if uid, ok := userIDVal.(uuid.UUID); ok {
		userIDStr = uid.String()
	}

	secret, err := h.service.InitiatePayment(id, userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":              "payment initiated successfully",
		"stripe_client_secret": secret,
	})
}

func (h *BookingHandler) CancelBooking(c fiber.Ctx) error {
	id := c.Params("bookingId")
	var req dto.CancelBookingRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	userIDVal := c.Locals("userID")
	userIDStr := ""
	if uid, ok := userIDVal.(uuid.UUID); ok {
		userIDStr = uid.String()
	}

	resp, err := h.service.CancelBooking(id, userIDStr, &req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "booking cancelled successfully",
		"data":    resp,
	})
}

func (h *BookingHandler) GetTicket(c fiber.Ctx) error {
	id := c.Params("bookingId")

	ticket, err := h.service.GetTicket(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "ticket not found"})
	}

	return c.JSON(ticket)
}
