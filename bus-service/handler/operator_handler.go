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
	opID, ok := c.Locals("operatorID").(uuid.UUID)
	if !ok {
		return utils.Fail(c, fiber.StatusForbidden, "Operator context missing")
	}

	profile, err := h.service.GetProfile(opID.String())
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

func (h *OperatorHandler) LoadInventory(c fiber.Ctx) error {
	opID, ok := c.Locals("operatorID").(uuid.UUID)
	if !ok {
		return utils.Fail(c, fiber.StatusForbidden, "Operator context missing")
	}

	var req dto.LoadInventoryReq
	if err := c.Bind().JSON(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "Invalid request body")
	}

	inv, err := h.service.LoadInventory(opID.String(), req)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusCreated, "Inventory loaded successfully", inv)
}

func (h *OperatorHandler) GetInventoryBookings(c fiber.Ctx) error {
	opID, ok := c.Locals("operatorID").(uuid.UUID)
	if !ok {
		return utils.Fail(c, fiber.StatusForbidden, "Operator context missing")
	}

	invID := c.Params("id")
	if invID == "" {
		return utils.Fail(c, fiber.StatusBadRequest, "Inventory ID is required")
	}

	bookings, err := h.service.GetInventoryBookings(opID.String(), invID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.Success(c, fiber.StatusOK, "Bookings retrieved successfully", bookings)
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
