package handler

import (
	"strconv"
	"github.com/gofiber/fiber/v3"
	
	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/Salman-kp/tripneo/bus-service/pkg/utils"
	"github.com/Salman-kp/tripneo/bus-service/service"
)

type BusHandler struct {
	service service.BusService
}

func NewBusHandler(service service.BusService) *BusHandler {
	return &BusHandler{service: service}
}

// 1. GET /api/buses/search
func (h *BusHandler) SearchBuses(c fiber.Ctx) error {
	origin := c.Query("origin")
	destination := c.Query("destination")
	travelDate := c.Query("travel_date") // e.g., 2026-04-15
	
	if travelDate == "" || origin == "" || destination == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "travel_date, origin, and destination are required fields")
	}

	passengers, _ := strconv.Atoi(c.Query("passengers"))
	minPrice, _ := strconv.ParseFloat(c.Query("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(c.Query("max_price"), 64)

	filter := model.SearchBusFilter{
		Origin:        origin,
		Destination:   destination,
		TravelDate:    travelDate,
		Passengers:    passengers,
		SeatType:      c.Query("seat_type"),
		Operator:      c.Query("operator"),
		MinPrice:      minPrice,
		MaxPrice:      maxPrice,
		SortBy:        c.Query("sort_by"),
		DepartureTime: c.Query("departure_time"),
	}

	instances, err := h.service.SearchBuses(filter)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Buses retrieved effectively", instances)
}

// 2. GET /api/buses/:instanceId
func (h *BusHandler) GetBus(c fiber.Ctx) error {
	id := c.Params("instanceId")
	
	instance, err := h.service.GetBusInstance(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "Bus instance not found cleanly")
	}

	return utils.Success(c, fiber.StatusOK, "Bus instance loaded successfully", instance)
}

// 3. GET /api/buses/:instanceId/fares
func (h *BusHandler) GetBusFares(c fiber.Ctx) error {
	id := c.Params("instanceId")
	
	fares, err := h.service.GetFares(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Fares extracted automatically", fares)
}

// 4. GET /api/buses/:instanceId/seats
func (h *BusHandler) GetBusSeats(c fiber.Ctx) error {
	id := c.Params("instanceId")
	
	seats, err := h.service.GetSeats(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Seat Layout fully aggregated", seats)
}

// 5. GET /api/buses/:instanceId/amenities
func (h *BusHandler) GetBusAmenities(c fiber.Ctx) error {
	id := c.Params("instanceId")
	
	amenities, err := h.service.GetAmenities(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, "Bus amenities failed to unpack gracefully")
	}

	return utils.Success(c, fiber.StatusOK, "Amenities discovered transparently", amenities)
}

// 6. GET /api/buses/:instanceId/boarding-points
func (h *BusHandler) GetBoardingPoints(c fiber.Ctx) error {
	id := c.Params("instanceId")
	
	points, err := h.service.GetBoardingPoints(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Iterated all structural transit stopping metrics completely", points)
}