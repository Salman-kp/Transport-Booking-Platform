package handler

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/dto"
	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/Salman-kp/tripneo/bus-service/pkg/utils"
	"github.com/Salman-kp/tripneo/bus-service/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type AdminHandler struct {
	service service.AdminService
}

func NewAdminHandler(service service.AdminService) *AdminHandler {
	return &AdminHandler{service: service}
}

func (h *AdminHandler) CreateBus(c fiber.Ctx) error {
	var bus model.Bus
	if err := c.Bind().JSON(&bus); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if err := h.service.CreateBus(&bus); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusCreated, "Bus created successfully", bus)
}

func (h *AdminHandler) UpdateBus(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid bus ID")
	}

	var updates map[string]interface{}
	if err := c.Bind().JSON(&updates); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if err := h.service.UpdateBus(id, updates); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Bus updated successfully", nil)
}

func (h *AdminHandler) ListBuses(c fiber.Ctx) error {
	buses, err := h.service.ListBuses()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Buses retrieved successfully", buses)
}

func (h *AdminHandler) GetPricingRules(c fiber.Ctx) error {
	rules, err := h.service.GetPricingRules()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Pricing rules retrieved successfully", rules)
}

func (h *AdminHandler) ApproveOperator(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid operator ID")
	}

	if err := h.service.UpdateOperatorStatus(id, "ACTIVE"); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Operator approved successfully", nil)
}

func (h *AdminHandler) SuspendOperator(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid operator ID")
	}

	if err := h.service.UpdateOperatorStatus(id, "SUSPENDED"); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Operator suspended successfully", nil)
}

func (h *AdminHandler) GetAllBookings(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	bookings, totalCount, err := h.service.ListAllBookings(page, limit)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Bookings retrieved successfully", fiber.Map{
		"bookings":    bookings,
		"total_count": totalCount,
		"page":        page,
		"limit":       limit,
	})
}

func (h *AdminHandler) GetRevenueAnalytics(c fiber.Ctx) error {
	analytics, err := h.service.GetRevenueAnalytics()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Revenue analytics retrieved successfully", analytics)
}

func (h *AdminHandler) UpdatePricingRule(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid pricing rule ID")
	}

	var updates map[string]interface{}
	if err := c.Bind().JSON(&updates); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if name, ok := updates["name"]; ok && name == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "name cannot be empty")
	}
	if rt, ok := updates["rule_type"]; ok && rt == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "rule_type cannot be empty")
	}
	if m, ok := updates["multiplier"]; ok {
		switch v := m.(type) {
		case float64:
			if v <= 0 {
				return utils.Fail(c, fiber.StatusBadRequest, "multiplier must be greater than 0")
			}
		case string:
			return utils.Fail(c, fiber.StatusBadRequest, "multiplier must be a number")
		}
	}

	if conditions, ok := updates["conditions"]; ok {
		bytes, err := json.Marshal(conditions)
		if err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "invalid conditions format")
		}
		updates["conditions"] = bytes
	}

	if err := h.service.UpdatePricingRule(id, updates); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Pricing rule updated successfully", nil)
}

func (h *AdminHandler) CreatePricingRule(c fiber.Ctx) error {
	var rule model.PricingRule
	if err := c.Bind().JSON(&rule); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if rule.Name == "" || rule.RuleType == "" || len(rule.Conditions) == 0 || rule.Multiplier <= 0 {
		return utils.Fail(c, fiber.StatusBadRequest, "Missing or invalid required fields (name, rule_type, conditions, multiplier > 0)")
	}

	if err := h.service.CreatePricingRule(&rule); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusCreated, "Pricing rule created successfully", rule)
}

func (h *AdminHandler) GetOperatorAnalytics(c fiber.Ctx) error {
	analytics, err := h.service.GetOperatorAnalytics()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Operator analytics retrieved successfully", analytics)
}

func (h *AdminHandler) GetUpcomingTrips(c fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 || limit > 1000 {
		limit = 1000
	}

	now := time.Now()
	month, _ := strconv.Atoi(c.Query("month", strconv.Itoa(int(now.Month()))))
	year, _ := strconv.Atoi(c.Query("year", strconv.Itoa(now.Year())))

	if month < 1 || month > 12 {
		month = int(now.Month())
	}
	if year < 2000 {
		year = now.Year()
	}

	trips, err := h.service.GetUpcomingTrips(limit, month, year)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Upcoming trips retrieved successfully", trips)
}

func (h *AdminHandler) CreateBusType(c fiber.Ctx) error {
	var busType model.BusType
	if err := c.Bind().JSON(&busType); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := h.service.CreateBusType(&busType); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusCreated, "Bus type created successfully", busType)
}

func (h *AdminHandler) GetBusTypes(c fiber.Ctx) error {
	types, err := h.service.GetBusTypes()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Bus types retrieved successfully", types)
}

func (h *AdminHandler) CreateBusStop(c fiber.Ctx) error {
	var stop model.BusStop
	if err := c.Bind().JSON(&stop); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := h.service.CreateBusStop(&stop); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusCreated, "Bus stop created successfully", stop)
}

func (h *AdminHandler) GetBusStops(c fiber.Ctx) error {
	stops, err := h.service.GetBusStops()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Bus stops retrieved successfully", stops)
}

func (h *AdminHandler) CreateOperator(c fiber.Ctx) error {
	var operator model.Operator
	if err := c.Bind().JSON(&operator); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if err := h.service.CreateOperator(&operator); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusCreated, "Operator created successfully", operator)
}

func (h *AdminHandler) GetOperators(c fiber.Ctx) error {
	operators, err := h.service.GetOperators()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Operators retrieved successfully", operators)
}

func (h *AdminHandler) DeleteBusInstance(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid instance ID")
	}

	if err := h.service.DeleteBusInstance(id); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Bus instance deleted successfully", nil)
}

func (h *AdminHandler) UpdateBusInstanceStatus(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid instance ID")
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	validStatuses := map[string]bool{
		"SCHEDULED": true,
		"COMPLETED": true,
		"CANCELLED": true,
		"DELAYED":   true,
	}
	if !validStatuses[req.Status] {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid status. Allowed values: SCHEDULED, COMPLETED, CANCELLED, DELAYED")
	}

	if err := h.service.UpdateBusInstanceStatus(id, req.Status); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Bus instance status updated successfully", nil)
}

func (h *AdminHandler) GetCancellationPolicies(c fiber.Ctx) error {
	policies, err := h.service.GetCancellationPolicies()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Cancellation policies retrieved successfully", policies)
}

func (h *AdminHandler) CreateCancellationPolicy(c fiber.Ctx) error {
	var policy model.CancellationPolicy
	if err := c.Bind().JSON(&policy); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if policy.Name == "" || policy.HoursBeforeDeparture < 0 || policy.RefundPercentage < 0 || policy.RefundPercentage > 100 || policy.CancellationFee < 0 {
		return utils.Fail(c, fiber.StatusBadRequest, "Missing or invalid required fields (RefundPercentage must be 0-100, Fee must be >= 0)")
	}

	if err := h.service.CreateCancellationPolicy(&policy); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusCreated, "Cancellation policy created successfully", policy)
}

func (h *AdminHandler) UpdateCancellationPolicy(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid cancellation policy ID")
	}

	var updates map[string]interface{}
	if err := c.Bind().JSON(&updates); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if name, ok := updates["name"]; ok && name == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "name cannot be empty")
	}

	if hours, ok := updates["hours_before_departure"]; ok {
		switch v := hours.(type) {
		case float64:
			if v < 0 {
				return utils.Fail(c, fiber.StatusBadRequest, "hours_before_departure cannot be negative")
			}
		case string:
			return utils.Fail(c, fiber.StatusBadRequest, "hours_before_departure must be a number")
		}
	}

	if refund, ok := updates["refund_percentage"]; ok {
		switch v := refund.(type) {
		case float64:
			if v < 0 || v > 100 {
				return utils.Fail(c, fiber.StatusBadRequest, "refund_percentage must be between 0 and 100")
			}
		case string:
			return utils.Fail(c, fiber.StatusBadRequest, "refund_percentage must be a number")
		}
	}

	if fee, ok := updates["cancellation_fee"]; ok {
		switch v := fee.(type) {
		case float64:
			if v < 0 {
				return utils.Fail(c, fiber.StatusBadRequest, "cancellation_fee cannot be negative")
			}
		case string:
			return utils.Fail(c, fiber.StatusBadRequest, "cancellation_fee must be a number")
		}
	}

	if err := h.service.UpdateCancellationPolicy(id, updates); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Cancellation policy updated successfully", nil)
}

func (h *AdminHandler) GetDailyAccountingAnalytics(c fiber.Ctx) error {
	monthStr := c.Query("month")
	yearStr := c.Query("year")

	if monthStr == "" || yearStr == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "month and year are required query parameters")
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid month parameter")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid year parameter")
	}

	analytics, err := h.service.GetDailyAccountingAnalytics(month, year)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Daily accounting analytics retrieved successfully", analytics)
}

func (h *AdminHandler) GetInstanceAccountingAnalytics(c fiber.Ctx) error {
	dayStr := c.Query("day")
	monthStr := c.Query("month")
	yearStr := c.Query("year")

	if dayStr == "" || monthStr == "" || yearStr == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "day, month, and year are required query parameters")
	}

	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid day parameter")
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid month parameter")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2000 {
		return utils.Fail(c, fiber.StatusBadRequest, "invalid year parameter")
	}

	analytics, err := h.service.GetInstanceAccountingAnalytics(day, month, year)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Instance accounting analytics retrieved successfully", analytics)
}

func (h *AdminHandler) GetBookingsByInstance(c fiber.Ctx) error {
	id := c.Params("id")
	bookings, err := h.service.GetBookingsByInstance(id)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}
	return utils.Success(c, fiber.StatusOK, "Instance bookings retrieved successfully", bookings)
}

func (h *AdminHandler) CancelBooking(c fiber.Ctx) error {
	id := c.Params("id")
	var req dto.CancelBookingRequest
	if err := c.Bind().JSON(&req); err != nil && err.Error() != "Unprocessable Entity" {
		// Ignore empty body errors, but fail on malformed JSON
		if len(c.Body()) > 0 {
			return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
		}
	}

	res, err := h.service.CancelBooking(id, &req)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Booking cancelled successfully", res)
}
