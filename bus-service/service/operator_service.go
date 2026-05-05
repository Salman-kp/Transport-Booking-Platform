package service

import (
	"errors"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/dto"
	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/google/uuid"
)

type OperatorService interface {
	RegisterOperator(req dto.RegisterOperatorReq) (*model.Operator, error)
	GetProfile(operatorID string) (*model.Operator, error)
	GetInventory(operatorID string) ([]model.OperatorInventory, error)
	LoadInventory(operatorID string, req dto.LoadInventoryReq) (*model.OperatorInventory, error)
	GetInventoryBookings(operatorID, inventoryID string) ([]model.Booking, error)
	GetAnalytics(operatorID string) (map[string]interface{}, error)
}

type operatorService struct {
	repo repository.OperatorRepository
}

func NewOperatorService(repo repository.OperatorRepository) OperatorService {
	return &operatorService{repo: repo}
}

func (s *operatorService) RegisterOperator(req dto.RegisterOperatorReq) (*model.Operator, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, errors.New("invalid user_id")
	}

	// 1. Create the operator
	op := model.Operator{
		Name:           req.Name,
		OperatorCode:   req.OperatorCode,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
		LogoURL:        req.LogoURL,
		CommissionRate: req.CommissionRate,
		Status:         "ACTIVE", // Auto-activate since admin is creating it
	}

	if err := s.repo.CreateOperator(&op); err != nil {
		return nil, err
	}

	// 2. Link user to operator
	opUser := model.OperatorUser{
		UserID:     userID,
		OperatorID: op.ID,
		Role:       "MANAGER",
		Status:     "ACTIVE",
	}

	if err := s.repo.CreateOperatorUser(&opUser); err != nil {
		return nil, err
	}

	return &op, nil
}

func (s *operatorService) GetProfile(operatorID string) (*model.Operator, error) {
	return s.repo.FindOperatorByID(operatorID)
}

func (s *operatorService) GetInventory(operatorID string) ([]model.OperatorInventory, error) {
	return s.repo.GetInventoryByOperator(operatorID)
}

func (s *operatorService) LoadInventory(operatorID string, req dto.LoadInventoryReq) (*model.OperatorInventory, error) {
	opUUID, err := uuid.Parse(operatorID)
	if err != nil {
		return nil, errors.New("invalid operator id")
	}

	busInstanceID, err := uuid.Parse(req.BusInstanceID)
	if err != nil {
		return nil, errors.New("invalid bus_instance_id")
	}

	fareTypeID, err := uuid.Parse(req.FareTypeID)
	if err != nil {
		return nil, errors.New("invalid fare_type_id")
	}

	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		return nil, errors.New("invalid expires_at format, expected RFC3339")
	}

	// VERIFY OWNERSHIP: Ensure the operator actually owns this bus instance
	owns, err := s.repo.VerifyBusInstanceOwnership(operatorID, req.BusInstanceID)
	if err != nil {
		return nil, err
	}
	if !owns {
		return nil, errors.New("forbidden: you do not own the requested bus instance")
	}

	inv := model.OperatorInventory{
		OperatorID:     opUUID,
		BusInstanceID:  busInstanceID,
		FareTypeID:     fareTypeID,
		SeatType:       req.SeatType,
		QuantityLoaded: req.QuantityLoaded,
		WholesalePrice: req.WholesalePrice,
		SellingPrice:   req.SellingPrice,
		Status:         "ACTIVE",
		ExpiresAt:      expiresAt,
	}

	if err := s.repo.LoadInventory(&inv); err != nil {
		return nil, err
	}

	return &inv, nil
}

func (s *operatorService) GetInventoryBookings(operatorID, inventoryID string) ([]model.Booking, error) {
	return s.repo.GetBookingsByInventory(inventoryID, operatorID)
}

func (s *operatorService) GetAnalytics(operatorID string) (map[string]interface{}, error) {
	return s.repo.GetOperatorAnalytics(operatorID)
}
