package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/flight-service/dto"
	"github.com/junaid9001/tripneo/flight-service/services"
)

type FlightHandler struct {
	flightService *services.FlightService
}

func NewFlightHandler(fs *services.FlightService) *FlightHandler {
	return &FlightHandler{flightService: fs}
}

// Search
func (h *FlightHandler) Search(c fiber.Ctx) error {
	var req dto.FlightSearchRequest

	if err := c.Bind().Query(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to parse query parameters cleanly",
		})
	}

	if req.Class == "" {
		req.Class = "ECONOMY"
	}
	if req.Passengers == 0 {
		req.Passengers = 1
	}

	results, err := h.flightService.SearchFlights(req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error occurred while searching for flights",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    results,
	})
}

func (h *FlightHandler) GetFlightDetails(c fiber.Ctx) error {
	id := c.Params("instanceId")
	res, err := h.flightService.GetFlightDetails(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": res})
}

func (h *FlightHandler) GetFares(c fiber.Ctx) error {
	id := c.Params("instanceId")
	res, err := h.flightService.GetFares(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": res})
}

func (h *FlightHandler) GetSeatMap(c fiber.Ctx) error {
	id := c.Params("instanceId")
	res, err := h.flightService.GetSeatMap(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true, "data": res})
}

func (h *FlightHandler) GetAncillaries(c fiber.Ctx) error {
	id := c.Params("instanceId")
	res, _ := h.flightService.GetAncillaries(id)
	return c.JSON(fiber.Map{"success": true, "data": res})
}

func (h *FlightHandler) GetFarePrediction(c fiber.Ctx) error {
	id := c.Params("instanceId")
	res, _ := h.flightService.GetFarePrediction(id)
	return c.JSON(fiber.Map{"success": true, "data": res})
}
