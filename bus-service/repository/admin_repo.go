package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminRepository interface {
	CreateBus(bus *model.Bus) error
	UpdateBus(id uuid.UUID, updates map[string]interface{}) error
	UpdateOperatorStatus(id uuid.UUID, updates map[string]interface{}) error
	ListAllBookings(page, limit int) ([]model.Booking, int64, error)
	GetConfirmedBookingsCount() (int64, error)
	GetConfirmedBookingsRevenue() (float64, error)
	UpdatePricingRule(id uuid.UUID, updates map[string]interface{}) error
	CreatePricingRule(rule *model.PricingRule) error
	GetCancellationPolicies() ([]model.CancellationPolicy, error)
	CreateCancellationPolicy(policy *model.CancellationPolicy) error
	UpdateCancellationPolicy(id uuid.UUID, updates map[string]interface{}) error
	GetOperatorAnalytics() ([]map[string]interface{}, error)
	GetUpcomingTrips(limit, month, year int) ([]model.BusInstance, error)
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
	GetDailyAccountingAnalytics(month, year int) ([]map[string]interface{}, error)
	GetInstanceAccountingAnalytics(day, month, year int) ([]map[string]interface{}, error)
	GetBookingsByInstance(instanceID string) ([]model.Booking, error)
}

type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) CreateBus(bus *model.Bus) error {
	return r.db.Create(bus).Error
}

func (r *adminRepository) UpdateBus(id uuid.UUID, updates map[string]interface{}) error {
	result := r.db.Model(&model.Bus{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("bus not found")
	}
	return nil
}

func (r *adminRepository) UpdateOperatorStatus(id uuid.UUID, updates map[string]interface{}) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Operator{}).Where("id = ?", id).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("operator not found")
		}

		// Cascade the is_active status to all buses owned by this operator
		if isActive, ok := updates["is_active"].(bool); ok {
			if err := tx.Model(&model.Bus{}).Where("operator_id = ?", id).Update("is_active", isActive).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *adminRepository) ListAllBookings(page, limit int) ([]model.Booking, int64, error) {
	var bookings []model.Booking
	var totalCount int64

	// Count total records
	if err := r.db.Model(&model.Booking{}).Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Fetch paginated records
	err := r.db.
		Preload("BusInstance").
		Preload("BusInstance.Bus.Operator").
		Preload("FareType").
		Preload("Passengers").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&bookings).Error

	return bookings, totalCount, err
}

func (r *adminRepository) GetConfirmedBookingsCount() (int64, error) {
	var totalBookings int64
	err := r.db.Model(&model.Booking{}).
		Where("status = ?", "CONFIRMED").
		Count(&totalBookings).Error
	return totalBookings, err
}

func (r *adminRepository) GetConfirmedBookingsRevenue() (float64, error) {
	var totalRevenue float64
	err := r.db.Model(&model.Booking{}).
		Where("status = ?", "CONFIRMED").
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&totalRevenue).Error
	return totalRevenue, err
}

func (r *adminRepository) UpdatePricingRule(id uuid.UUID, updates map[string]interface{}) error {
	result := r.db.Model(&model.PricingRule{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("pricing rule not found")
	}
	return nil
}

func (r *adminRepository) CreatePricingRule(rule *model.PricingRule) error {
	return r.db.Create(rule).Error
}

func (r *adminRepository) GetCancellationPolicies() ([]model.CancellationPolicy, error) {
	var policies []model.CancellationPolicy
	err := r.db.Order("hours_before_departure DESC").Find(&policies).Error
	return policies, err
}

func (r *adminRepository) CreateCancellationPolicy(policy *model.CancellationPolicy) error {
	return r.db.Create(policy).Error
}

func (r *adminRepository) UpdateCancellationPolicy(id uuid.UUID, updates map[string]interface{}) error {
	result := r.db.Model(&model.CancellationPolicy{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("cancellation policy not found")
	}
	return nil
}

func (r *adminRepository) GetOperatorAnalytics() ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.db.Table("operators").
		Select(`
			operators.id, 
			operators.name, 
			COUNT(CASE WHEN bookings.status = 'CONFIRMED' THEN 1 END) as total_bookings,
			COUNT(CASE WHEN bookings.status = 'CANCELLED' THEN 1 END) as total_cancellations
		`).
		Joins("LEFT JOIN buses ON buses.operator_id = operators.id").
		Joins("LEFT JOIN bus_instances ON bus_instances.bus_id = buses.id").
		Joins("LEFT JOIN bookings ON bookings.bus_instance_id = bus_instances.id").
		Group("operators.id, operators.name").
		Order("total_bookings DESC").
		Find(&results).Error
	return results, err
}

func (r *adminRepository) GetUpcomingTrips(limit, month, year int) ([]model.BusInstance, error) {
	var instances []model.BusInstance
	err := r.db.Preload("Bus").Preload("Bus.Operator").
		Preload("Bus.OriginStop").Preload("Bus.DestinationStop").
		Where("EXTRACT(MONTH FROM departure_at) = ? AND EXTRACT(YEAR FROM departure_at) = ?", month, year).
		Order("departure_at ASC").
		Limit(limit).
		Find(&instances).Error
	return instances, err
}

func (r *adminRepository) CreateBusType(busType *model.BusType) error {
	return r.db.Create(busType).Error
}

func (r *adminRepository) ListBuses() ([]model.Bus, error) {
	var buses []model.Bus
	err := r.db.Preload("Operator").Preload("BusType").
		Preload("OriginStop").Preload("DestinationStop").
		Find(&buses).Error
	return buses, err
}

func (r *adminRepository) GetBusTypes() ([]model.BusType, error) {
	var busTypes []model.BusType
	err := r.db.Find(&busTypes).Error
	return busTypes, err
}

func (r *adminRepository) GetPricingRules() ([]model.PricingRule, error) {
	var rules []model.PricingRule
	err := r.db.Order("priority desc").Find(&rules).Error
	return rules, err
}

func (r *adminRepository) CreateBusStop(busStop *model.BusStop) error {
	return r.db.Create(busStop).Error
}

func (r *adminRepository) GetBusStops() ([]model.BusStop, error) {
	var busStops []model.BusStop
	err := r.db.Find(&busStops).Error
	return busStops, err
}

func (r *adminRepository) CreateOperator(operator *model.Operator) error {
	return r.db.Create(operator).Error
}

func (r *adminRepository) GetOperators() ([]model.Operator, error) {
	var operators []model.Operator
	err := r.db.Find(&operators).Error
	return operators, err
}

func (r *adminRepository) DeleteBusInstance(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Delete associated records first to avoid FK constraint violations
		if err := tx.Where("bus_instance_id = ?", id).Delete(&model.BoardingPoint{}).Error; err != nil {
			return err
		}
		if err := tx.Where("bus_instance_id = ?", id).Delete(&model.DroppingPoint{}).Error; err != nil {
			return err
		}
		if err := tx.Where("bus_instance_id = ?", id).Delete(&model.Seat{}).Error; err != nil {
			return err
		}
		if err := tx.Where("bus_instance_id = ?", id).Delete(&model.FareType{}).Error; err != nil {
			return err
		}

		// 2. Finally delete the instance
		result := tx.Delete(&model.BusInstance{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("bus instance not found")
		}
		return nil
	})
}

func (r *adminRepository) UpdateBusInstanceStatus(id uuid.UUID, status string) error {
	return r.db.Model(&model.BusInstance{}).Where("id = ?", id).Update("status", status).Error
}

func (r *adminRepository) GetDailyAccountingAnalytics(month, year int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.db.Model(&model.PrebookingAccounting{}).
		Select("DATE(created_at) as date, COALESCE(SUM(spend_amount_total), 0) as total_spend, COALESCE(SUM(profit_amount), 0) as total_profit, COALESCE(SUM(loss_amount), 0) as total_loss").
		Where("EXTRACT(MONTH FROM created_at) = ? AND EXTRACT(YEAR FROM created_at) = ?", month, year).
		Group("DATE(created_at)").
		Order("date DESC").
		Find(&results).Error
	return results, err
}

func (r *adminRepository) GetInstanceAccountingAnalytics(day, month, year int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := r.db.Model(&model.PrebookingAccounting{}).
		Select("instance_id, bus_number, spend_amount_total, profit_amount, loss_amount").
		Where("EXTRACT(DAY FROM created_at) = ? AND EXTRACT(MONTH FROM created_at) = ? AND EXTRACT(YEAR FROM created_at) = ?", day, month, year).
		Order("created_at DESC").
		Find(&results).Error
	return results, err
}

func (r *adminRepository) GetBookingsByInstance(instanceID string) ([]model.Booking, error) {
	var bookings []model.Booking
	err := r.db.
		Preload("BusInstance").
		Preload("BusInstance.Bus.Operator").
		Preload("FareType").
		Preload("Passengers").
		Where("bus_instance_id = ?", instanceID).
		Order("created_at DESC").
		Find(&bookings).Error
	return bookings, err
}
