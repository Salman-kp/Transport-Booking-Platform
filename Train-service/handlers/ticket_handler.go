package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nabeel-mp/tripneo/train-service/service"
)

// GetTicket handles GET /api/train/tickets/:booking_id
func GetTicket() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, _ := c.Locals("userID").(string)
		bookingID := c.Params("booking_id")

		ticket, err := service.GetTicket(bookingID, userID)
		if err != nil {
			return c.Status(mapDomainErrorToStatus(err)).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(200).JSON(ticket)
	}
}

// VerifyTicket handles POST /api/train/tickets/verify
func VerifyTicket() fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			BookingID string `json:"booking_id" validate:"required"`
			Token     string `json:"token"      validate:"required"`
		}
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
		}

		result, err := service.VerifyTicket(req.BookingID, req.Token)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(200).JSON(result)
	}
}
