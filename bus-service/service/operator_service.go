package service

import (
	"errors"
	"github.com/Salman-kp/tripneo/bus-service/dto"
	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/google/uuid"
)

type OperatorService interface {
	RegisterOperator(req dto.RegisterOperatorReq) (*model.Operator, error)
	GetProfile(userID string) (map[string]interface{}, error)
	GetInventory(operatorID string) ([]model.OperatorInventory, error)
	GetAnalytics(operatorID string) (map[string]interface{}, error)
	GetBookingsByOperator(operatorID string) ([]model.Booking, error)
	
	// Trip Instance Management
	ListInstances(operatorID string) ([]model.BusInstance, error)
	RemoveInstance(operatorID, instanceID string) error
	ChangeInstanceStatus(operatorID, instanceID, status string) error
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

	var op model.Operator
	err = s.repo.WithTransaction(func(repo repository.OperatorRepository) error {
		// 1. Create the operator
		op = model.Operator{
			Name:           req.Name,
			OperatorCode:   req.OperatorCode,
			ContactEmail:   req.ContactEmail,
			ContactPhone:   req.ContactPhone,
			LogoURL:        req.LogoURL,
			CommissionRate: req.CommissionRate,
			Status:         "ACTIVE",
		}

		if err := repo.CreateOperator(&op); err != nil {
			return err
		}

		// 2. Link user to operator
		opUser := model.OperatorUser{
			UserID:     userID,
			OperatorID: op.ID,
			Role:       "MANAGER",
			Status:     "ACTIVE",
		}

		return repo.CreateOperatorUser(&opUser)
	})

	if err != nil {
		return nil, err
	}

	return &op, nil
}

func (s *operatorService) GetProfile(userID string) (map[string]interface{}, error) {
	opUser, op, err := s.repo.GetOperatorUserByUserID(userID)
	if err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"operator_user": opUser,
		"operator":      op,
	}, nil
}

func (s *operatorService) GetInventory(operatorID string) ([]model.OperatorInventory, error) {
	return s.repo.GetInventoryByOperator(operatorID)
}

func (s *operatorService) GetAnalytics(operatorID string) (map[string]interface{}, error) {
	return s.repo.GetOperatorAnalytics(operatorID)
}

func (s *operatorService) GetBookingsByOperator(operatorID string) ([]model.Booking, error) {
	return s.repo.GetBookingsByOperator(operatorID)
}

// Trip Instance Management

func (s *operatorService) ListInstances(operatorID string) ([]model.BusInstance, error) {
	return s.repo.GetInstancesByOperator(operatorID)
}

func (s *operatorService) RemoveInstance(operatorID, instanceID string) error {
	return s.repo.DeleteBusInstance(operatorID, instanceID)
}

func (s *operatorService) ChangeInstanceStatus(operatorID, instanceID, status string) error {
	return s.repo.UpdateInstanceStatus(operatorID, instanceID, status)
}
