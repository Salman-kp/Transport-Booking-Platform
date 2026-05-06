package handler

import (
	"github.com/Salman-kp/tripneo/bus-service/dto"
	"github.com/Salman-kp/tripneo/bus-service/pkg/utils"
	"github.com/Salman-kp/tripneo/bus-service/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type OperatorHandler struct {
	service service.OperatorService
}

func NewOperatorHandler(service service.OperatorService) *OperatorHandler {
	return &OperatorHandler{service: service}
}

func (h *OperatorHandler) RegisterOperator(c fiber.Ctx) error {
	var req dto.RegisterOperatorReq
	if err := c.Bind().JSON(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	op, err := h.service.RegisterOperator(req)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusCreated, "Operator registered successfully", op)
}

func (h *OperatorHandler) GetProfile(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return utils.Fail(c, fiber.StatusUnauthorized, "User context missing")
	}

	profile, err := h.service.GetProfile(userID.String())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Profile retrieved successfully", profile)
}

func (h *OperatorHandler) GetInventory(c fiber.Ctx) error {
	opID, ok := c.Locals("operatorID").(uuid.UUID)
	if !ok {
		return utils.Fail(c, fiber.StatusForbidden, "Operator context missing")
	}

	inventory, err := h.service.GetInventory(opID.String())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Inventory retrieved successfully", inventory)
}

func (h *OperatorHandler) GetAnalytics(c fiber.Ctx) error {
	opID, ok := c.Locals("operatorID").(uuid.UUID)
	if !ok {
		return utils.Fail(c, fiber.StatusForbidden, "Operator context missing")
	}

	analytics, err := h.service.GetAnalytics(opID.String())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Analytics retrieved successfully", analytics)
}

func (h *OperatorHandler) GetBookings(c fiber.Ctx) error {
	opID, ok := c.Locals("operatorID").(uuid.UUID)
	if !ok {
		return utils.Fail(c, fiber.StatusForbidden, "Operator context missing")
	}

	bookings, err := h.service.GetBookingsByOperator(opID.String())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Bookings retrieved successfully", bookings)
}

// ── Trip Instance Management ──────────────────────────────────────────────────

func (h *OperatorHandler) GetInstances(c fiber.Ctx) error {
	opID, ok := c.Locals("operatorID").(uuid.UUID)
	if !ok {
		return utils.Fail(c, fiber.StatusForbidden, "Operator context missing")
	}

	instances, err := h.service.ListInstances(opID.String())
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Instances retrieved successfully", instances)
}

func (h *OperatorHandler) DeleteInstance(c fiber.Ctx) error {
	opID, ok := c.Locals("operatorID").(uuid.UUID)
	if !ok {
		return utils.Fail(c, fiber.StatusForbidden, "Operator context missing")
	}

	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid instance ID format")
	}

	if err := h.service.RemoveInstance(opID.String(), idStr); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Instance deleted successfully", nil)
}

func (h *OperatorHandler) UpdateInstanceStatus(c fiber.Ctx) error {
	opID, ok := c.Locals("operatorID").(uuid.UUID)
	if !ok {
		return utils.Fail(c, fiber.StatusForbidden, "Operator context missing")
	}

	idStr := c.Params("id")
	if _, err := uuid.Parse(idStr); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid instance ID format")
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	// Mirroring Admin validation logic
	validStatuses := map[string]bool{
		"SCHEDULED": true,
		"COMPLETED": true,
		"CANCELLED": true,
		"DELAYED":   true,
	}
	if !validStatuses[req.Status] {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid status. Allowed values: SCHEDULED, COMPLETED, CANCELLED, DELAYED")
	}

	if err := h.service.ChangeInstanceStatus(opID.String(), idStr, req.Status); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Instance status updated successfully", nil)
}
