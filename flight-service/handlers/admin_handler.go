package handlers

import (
	"errors"
	"math"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/junaid9001/tripneo/flight-service/dto"
	"github.com/junaid9001/tripneo/flight-service/models"
	"github.com/junaid9001/tripneo/flight-service/services"
	"github.com/lib/pq"
)

type AdminHandler struct {
	service *services.AdminService
}

func NewAdminHandler(service *services.AdminService) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) GetAllBookings(c fiber.Ctx) error {
	bookings, err := h.service.ListAllBookings()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"bookings": bookings})
}

func (h *AdminHandler) UpdateBookingStatus(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid booking id"})
	}

	type Request struct {
		Status string `json:"status"`
	}
	var req Request
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "bad request"})
	}

	if err := h.service.ForceUpdateBookingStatus(id, req.Status); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *AdminHandler) CreateFlight(c fiber.Ctx) error {
	var req dto.CreateFlightRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "bad request"})
	}

	airlineID, err := uuid.Parse(req.AirlineID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid airline_id"})
	}

	aircraftTypeID, err := uuid.Parse(req.AircraftTypeID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid aircraft_type_id"})
	}

	originAirportID, err := uuid.Parse(req.OriginAirportID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid origin_airport_id"})
	}

	destinationAirportID, err := uuid.Parse(req.DestinationAirportID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid destination_airport_id"})
	}

	departureTime, err := time.Parse(time.RFC3339, req.DepartureTime)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid departure_time; expected RFC3339 format"})
	}

	arrivalTime, err := time.Parse(time.RFC3339, req.ArrivalTime)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid arrival_time; expected RFC3339 format"})
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	flight := models.Flight{
		FlightNumber:         req.FlightNumber,
		AirlineID:            airlineID,
		AircraftTypeID:       aircraftTypeID,
		OriginAirportID:      originAirportID,
		DestinationAirportID: destinationAirportID,
		DepartureTime:        departureTime,
		ArrivalTime:          arrivalTime,
		DurationMinutes:      req.DurationMinutes,
		DaysOfWeek:           pq.Int64Array(req.DaysOfWeek),
		IsActive:             isActive,
	}

	if err := h.service.CreateFlight(&flight); err != nil {
		switch {
		case errors.Is(err, services.ErrInvalidAirlineID),
			errors.Is(err, services.ErrInvalidAircraftTypeID),
			errors.Is(err, services.ErrInvalidOriginAirportID),
			errors.Is(err, services.ErrInvalidDestinationAirportID),
			errors.Is(err, services.ErrInvalidRoute):
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.Status(201).JSON(fiber.Map{"success": true, "flight": flight})
}

func (h *AdminHandler) UpdateFlight(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid flight id"})
	}

	var updates map[string]interface{}
	if err := c.Bind().JSON(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "bad request"})
	}

	if rawDays, exists := updates["days_of_week"]; exists {
		parsedDays, err := parseDaysOfWeek(rawDays)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid days_of_week; expected array of integers"})
		}
		updates["days_of_week"] = pq.Int64Array(parsedDays)
	}

	if rawDays, exists := updates["DaysOfWeek"]; exists {
		parsedDays, err := parseDaysOfWeek(rawDays)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid DaysOfWeek; expected array of integers"})
		}
		updates["days_of_week"] = pq.Int64Array(parsedDays)
		delete(updates, "DaysOfWeek")
	}

	if err := h.service.UpdateFlight(id, updates); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func parseDaysOfWeek(raw interface{}) ([]int64, error) {
	daySlice, ok := raw.([]interface{})
	if !ok {
		if typed, ok := raw.([]int64); ok {
			return typed, nil
		}
		return nil, fiber.ErrBadRequest
	}

	out := make([]int64, 0, len(daySlice))
	for _, d := range daySlice {
		switch v := d.(type) {
		case float64:
			if math.Trunc(v) != v {
				return nil, fiber.ErrBadRequest
			}
			out = append(out, int64(v))
		case int:
			out = append(out, int64(v))
		case int64:
			out = append(out, v)
		default:
			return nil, fiber.ErrBadRequest
		}
	}

	return out, nil
}

func (h *AdminHandler) DeleteFlight(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid flight id"})
	}

	if err := h.service.SoftDeleteFlight(id); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}

func (h *AdminHandler) UpdateFares(c fiber.Ctx) error {
	instanceIdStr := c.Params("instanceId")
	instanceId, err := uuid.Parse(instanceIdStr)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid instance id"})
	}

	type Request struct {
		EconomyPrice  float64 `json:"economy_price"`
		BusinessPrice float64 `json:"business_price"`
	}
	var req Request
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "bad request"})
	}

	if err := h.service.OverridePrices(instanceId, req.EconomyPrice, req.BusinessPrice); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"success": true})
}
