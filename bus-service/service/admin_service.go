package service

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminService interface {
	CreateBus(bus *model.Bus) error
	UpdateBus(id uuid.UUID, updates map[string]interface{}) error
	UpdateOperatorStatus(id uuid.UUID, status string) error
	ListAllBookings(page, limit int) ([]model.Booking, int64, error)
	GetRevenueAnalytics() (map[string]interface{}, error)
	UpdatePricingRule(id uuid.UUID, updates map[string]interface{}) error
	CreatePricingRule(rule *model.PricingRule) error
	GetCancellationPolicies() ([]model.CancellationPolicy, error)
	CreateCancellationPolicy(policy *model.CancellationPolicy) error
	UpdateCancellationPolicy(id uuid.UUID, updates map[string]interface{}) error
	GetOperatorAnalytics() ([]map[string]interface{}, error)
	GetUpcomingTrips(limit int) ([]model.BusInstance, error)
	CreateBusType(busType *model.BusType) error
	GetBusTypes() ([]model.BusType, error)
	CreateBusStop(busStop *model.BusStop) error
	GetBusStops() ([]model.BusStop, error)
	CreateOperator(operator *model.Operator) error
	GetOperators() ([]model.Operator, error)
	DeleteBusInstance(id uuid.UUID) error
	UpdateBusInstanceStatus(id uuid.UUID, status string) error
	ListBuses() ([]model.Bus, error)
	GetPricingRules() ([]model.PricingRule, error)
}

type adminService struct {
	repo repository.AdminRepository
	db   *gorm.DB
}

func NewAdminService(repo repository.AdminRepository, db *gorm.DB) AdminService {
	return &adminService{repo: repo, db: db}
}

func (s *adminService) CreateBus(bus *model.Bus) error {
	// 1. Basic Validation
	if bus.BusNumber == "" {
		return errors.New("bus number is required")
	}
	if bus.OperatorID == uuid.Nil {
		return errors.New("operator id is required")
	}
	if bus.BusTypeID == uuid.Nil {
		return errors.New("bus type id is required")
	}
	if bus.OriginStopID == uuid.Nil {
		return errors.New("origin stop id is required")
	}
	if bus.DestinationStopID == uuid.Nil {
		return errors.New("destination stop id is required")
	}
	if bus.DepartureTime == "" || bus.ArrivalTime == "" {
		return errors.New("departure and arrival times are required")
	}
	if bus.OriginStopID == bus.DestinationStopID {
		return errors.New("origin and destination stops cannot be the same")
	}

	// 2. Verify Foreign Keys Exist
	var count int64
	if err := s.db.Model(&model.Operator{}).Where("id = ?", bus.OperatorID).Count(&count).Error; err != nil || count == 0 {
		return errors.New("invalid operator id")
	}
	if err := s.db.Model(&model.BusType{}).Where("id = ?", bus.BusTypeID).Count(&count).Error; err != nil || count == 0 {
		return errors.New("invalid bus type id")
	}
	if err := s.db.Model(&model.BusStop{}).Where("id = ?", bus.OriginStopID).Count(&count).Error; err != nil || count == 0 {
		return errors.New("invalid origin stop id")
	}
	if err := s.db.Model(&model.BusStop{}).Where("id = ?", bus.DestinationStopID).Count(&count).Error; err != nil || count == 0 {
		return errors.New("invalid destination stop id")
	}

	// 3. Validate DaysOfWeek
	var days []int
	if err := json.Unmarshal(bus.DaysOfWeek, &days); err != nil || len(days) == 0 {
		return errors.New("at least one valid day of week (1-7) must be selected")
	}

	// 4. Calculate DurationMinutes if not provided
	if bus.DurationMinutes == 0 && bus.DepartureTime != "" && bus.ArrivalTime != "" {
		dep, err1 := time.Parse("15:04", bus.DepartureTime)
		arr, err2 := time.Parse("15:04", bus.ArrivalTime)
		if err1 == nil && err2 == nil {
			duration := arr.Sub(dep).Minutes()
			if duration <= 0 {
				duration += 1440 // Add 24 hours for overnight trips
			}
			bus.DurationMinutes = int(duration)
		}
	}

	return s.repo.CreateBus(bus)
}

func (s *adminService) UpdateBus(id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no updates provided")
	}

	// 1. Verify foreign keys if they are being updated
	fieldsToCheck := map[string]interface{}{
		"operator_id":         &model.Operator{},
		"bus_type_id":         &model.BusType{},
		"origin_stop_id":      &model.BusStop{},
		"destination_stop_id": &model.BusStop{},
	}

	for field, modelType := range fieldsToCheck {
		if val, ok := updates[field]; ok {
			idStr, ok := val.(string)
			if !ok {
				return errors.New("invalid format for " + field)
			}
			parsedID, err := uuid.Parse(idStr)
			if err != nil {
				return errors.New("invalid uuid format for " + field)
			}
			var count int64
			if err := s.db.Model(modelType).Where("id = ?", parsedID).Count(&count).Error; err != nil || count == 0 {
				return errors.New(field + " record not found")
			}
		}
	}

	return s.repo.UpdateBus(id, updates)
}

func (s *adminService) UpdateOperatorStatus(id uuid.UUID, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	switch status {
	case "ACTIVE":
		updates["is_active"] = true
	case "SUSPENDED":
		updates["is_active"] = false
	}

	return s.repo.UpdateOperatorStatus(id, updates)
}

func (s *adminService) ListAllBookings(page, limit int) ([]model.Booking, int64, error) {
	return s.repo.ListAllBookings(page, limit)
}

func (s *adminService) GetRevenueAnalytics() (map[string]interface{}, error) {
	totalBookings, err := s.repo.GetConfirmedBookingsCount()
	if err != nil {
		return nil, err
	}

	totalRevenue, err := s.repo.GetConfirmedBookingsRevenue()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_revenue":  totalRevenue,
		"total_bookings": totalBookings,
	}, nil
}

func (s *adminService) UpdatePricingRule(id uuid.UUID, updates map[string]interface{}) error {
	return s.repo.UpdatePricingRule(id, updates)
}

func (s *adminService) CreatePricingRule(rule *model.PricingRule) error {
	if rule.Name == "" || rule.RuleType == "" {
		return errors.New("pricing rule name and rule_type are required")
	}
	return s.repo.CreatePricingRule(rule)
}

func (s *adminService) GetCancellationPolicies() ([]model.CancellationPolicy, error) {
	return s.repo.GetCancellationPolicies()
}

func (s *adminService) CreateCancellationPolicy(policy *model.CancellationPolicy) error {
	if policy.Name == "" {
		return errors.New("cancellation policy name is required")
	}
	if policy.HoursBeforeDeparture < 0 {
		return errors.New("hours before departure cannot be negative")
	}
	return s.repo.CreateCancellationPolicy(policy)
}

func (s *adminService) UpdateCancellationPolicy(id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return errors.New("no updates provided")
	}
	return s.repo.UpdateCancellationPolicy(id, updates)
}

func (s *adminService) GetOperatorAnalytics() ([]map[string]interface{}, error) {
	return s.repo.GetOperatorAnalytics()
}

func (s *adminService) GetUpcomingTrips(limit int) ([]model.BusInstance, error) {
	return s.repo.GetUpcomingTrips(limit)
}

func (s *adminService) CreateBusType(busType *model.BusType) error {
	if busType.Name == "" {
		return errors.New("bus type name is required")
	}
	return s.repo.CreateBusType(busType)
}

func (s *adminService) GetBusTypes() ([]model.BusType, error) {
	return s.repo.GetBusTypes()
}

func (s *adminService) CreateBusStop(busStop *model.BusStop) error {
	if busStop.Name == "" || busStop.City == "" {
		return errors.New("bus stop name and city are required")
	}
	return s.repo.CreateBusStop(busStop)
}

func (s *adminService) GetBusStops() ([]model.BusStop, error) {
	return s.repo.GetBusStops()
}

func (s *adminService) CreateOperator(operator *model.Operator) error {
	if operator.Name == "" || operator.ContactEmail == "" {
		return errors.New("operator name and email are required")
	}
	return s.repo.CreateOperator(operator)
}

func (s *adminService) GetOperators() ([]model.Operator, error) {
	return s.repo.GetOperators()
}

func (s *adminService) DeleteBusInstance(id uuid.UUID) error {
	return s.repo.DeleteBusInstance(id)
}

func (s *adminService) UpdateBusInstanceStatus(id uuid.UUID, status string) error {
	return s.repo.UpdateBusInstanceStatus(id, status)
}

func (s *adminService) ListBuses() ([]model.Bus, error) {
	return s.repo.ListBuses()
}

func (s *adminService) GetPricingRules() ([]model.PricingRule, error) {
	return s.repo.GetPricingRules()
}
