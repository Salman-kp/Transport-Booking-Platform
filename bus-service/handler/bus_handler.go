package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Salman-kp/tripneo/bus-service/dto"
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

// helper: validate UUID
func validateUUID(c fiber.Ctx, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid instance ID")
	}
	return nil
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
	if instances == nil {
		instances = []model.BusInstance{}
	}

	return utils.Success(c, fiber.StatusOK, "Buses retrieved successfully", instances)
}

// 2. GET /api/v1/buses/:instanceId
func (h *BusHandler) GetBus(c fiber.Ctx) error {
	id := c.Params("instanceId")

	if err := validateUUID(c, id); err != nil {
		return err
	}

	instance, err := h.service.GetBusInstance(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.Fail(c, fiber.StatusNotFound, "Bus instance not found")
		}
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Bus instance loaded successfully", instance)
}

// 3. GET /api/buses/:instanceId/fares
func (h *BusHandler) GetBusFares(c fiber.Ctx) error {
	id := c.Params("instanceId")

	if err := validateUUID(c, id); err != nil {
		return err
	}

	fares, err := h.service.GetFares(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	if fares == nil {
		fares = []dto.FareResponse{}
	}

	return utils.Success(c, fiber.StatusOK, "Fares retrieved successfully", fares)
}

// 4. GET /api/buses/:instanceId/seats
func (h *BusHandler) GetBusSeats(c fiber.Ctx) error {
	id := c.Params("instanceId")

	if err := validateUUID(c, id); err != nil {
		return err
	}

	seats, err := h.service.GetSeats(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	if seats == nil {
		seats = []dto.SeatResponse{}
	}

	return utils.Success(c, fiber.StatusOK, "Seats retrieved successfully", seats)
}

// 5. GET /api/v1/buses/:instanceId/amenities
func (h *BusHandler) GetBusAmenities(c fiber.Ctx) error {
	id := c.Params("instanceId")

	if err := validateUUID(c, id); err != nil {
		return err
	}

	amenities, err := h.service.GetAmenities(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Amenities retrieved successfully", amenities)
}

// 6. GET /api/buses/:instanceId/boarding-points
func (h *BusHandler) GetBoardingPoints(c fiber.Ctx) error {
	id := c.Params("instanceId")

	if err := validateUUID(c, id); err != nil {
		return err
	}

	points, err := h.service.GetBoardingPoints(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Boarding points retrieved successfully", points)
}

// 7. GET /api/buses/:instanceId/dropping-points
func (h *BusHandler) GetDroppingPoints(c fiber.Ctx) error {
	id := c.Params("instanceId")

	if err := validateUUID(c, id); err != nil {
		return err
	}

	points, err := h.service.GetDroppingPoints(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Dropping points retrieved successfully", points)
}

// 8. GET /api/buses/:instanceId/route
func (h *BusHandler) GetRoute(c fiber.Ctx) error {
	id := c.Params("instanceId")

	if err := validateUUID(c, id); err != nil {
		return err
	}

	route, err := h.service.GetRoute(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Route retrieved successfully", route)
}

// 9. GET /api/bus/bus-stops?search=
func (h *BusHandler) GetBusStops(c fiber.Ctx) error {
	search := c.Query("search")

	stops, err := h.service.GetBusStops(search)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	if stops == nil {
		stops = []model.BusStop{}
	}

	return utils.Success(c, fiber.StatusOK, "Bus stops retrieved successfully", stops)
}

// 10. GET /api/bus/operators
func (h *BusHandler) GetOperators(c fiber.Ctx) error {
	operators, err := h.service.GetOperators()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	if operators == nil {
		operators = []model.Operator{}
	}

	return utils.Success(c, fiber.StatusOK, "Operators retrieved successfully", operators)
}
