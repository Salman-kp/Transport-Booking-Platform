package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OperatorRepository interface {
	CreateOperator(op *model.Operator) error
	CreateOperatorUser(opUser *model.OperatorUser) error
	GetOperatorUserByUserID(userID string) (*model.OperatorUser, *model.Operator, error)
	GetInventoryByOperator(operatorID string) ([]model.OperatorInventory, error)
	GetBookingsByOperator(operatorID string) ([]model.Booking, error)
	VerifyBusInstanceOwnership(operatorID, busInstanceID string) (bool, error)
	GetOperatorAnalytics(operatorID string) (map[string]interface{}, error)
	WithTransaction(fn func(repo OperatorRepository) error) error
	FindOperatorByID(id string) (*model.Operator, error)

	// Trip Instance Management
	GetInstancesByOperator(opID string) ([]model.BusInstance, error)
	DeleteBusInstance(opID, instanceID string) error
	UpdateInstanceStatus(opID, instanceID, status string) error
}

type operatorRepository struct {
	db *gorm.DB
}

func NewOperatorRepository(db *gorm.DB) OperatorRepository {
	return &operatorRepository{db: db}
}

func (r *operatorRepository) CreateOperator(op *model.Operator) error {
	return r.db.Create(op).Error
}

func (r *operatorRepository) CreateOperatorUser(opUser *model.OperatorUser) error {
	return r.db.Create(opUser).Error
}

func (r *operatorRepository) GetOperatorUserByUserID(userID string) (*model.OperatorUser, *model.Operator, error) {
	var opUser model.OperatorUser
	err := r.db.Preload("Operator").Where("user_id = ?", userID).First(&opUser).Error
	if err != nil {
		return nil, nil, err
	}
	return &opUser, &opUser.Operator, nil
}

func (r *operatorRepository) GetInventoryByOperator(operatorID string) ([]model.OperatorInventory, error) {
	var inventories []model.OperatorInventory
	err := r.db.Preload("BusInstance.Bus").Preload("FareType").Where("operator_id = ?", operatorID).Find(&inventories).Error
	return inventories, err
}

func (r *operatorRepository) VerifyBusInstanceOwnership(operatorID, busInstanceID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.BusInstance{}).
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("bus_instances.id = ? AND buses.operator_id = ?", busInstanceID, operatorID).
		Count(&count).Error
	return count > 0, err
}

func (r *operatorRepository) GetBookingsByOperator(operatorID string) ([]model.Booking, error) {
	var bookings []model.Booking
	err := r.db.
		Joins("JOIN bus_instances ON bus_instances.id = bookings.bus_instance_id").
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("buses.operator_id = ?", operatorID).
		Preload("Passengers").
		Preload("BusInstance.Bus").
		Preload("FareType").
		Find(&bookings).Error
	return bookings, err
}

func (r *operatorRepository) GetOperatorAnalytics(operatorID string) (map[string]interface{}, error) {
	var totalBookings int64
	var totalTicketsSold int64

	// 1. Gross Revenue & Bookings count (CONFIRMED only)
	err := r.db.Model(&model.Booking{}).
		Joins("JOIN bus_instances ON bus_instances.id = bookings.bus_instance_id").
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("bookings.status = ? AND buses.operator_id = ?", "CONFIRMED", operatorID).
		Count(&totalBookings).Error
	if err != nil {
		return nil, err
	}

	// Calculate total gross amount (what customers paid)
	var grossAmount float64
	err = r.db.Model(&model.Booking{}).
		Joins("JOIN bus_instances ON bus_instances.id = bookings.bus_instance_id").
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("bookings.status = ? AND buses.operator_id = ?", "CONFIRMED", operatorID).
		Select("COALESCE(SUM(bookings.total_amount), 0)").
		Scan(&grossAmount).Error
	if err != nil {
		return nil, err
	}

	// Calculate total base fare (the actual earnings basis)
	var totalBaseFare float64
	err = r.db.Model(&model.Booking{}).
		Joins("JOIN bus_instances ON bus_instances.id = bookings.bus_instance_id").
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("bookings.status = ? AND buses.operator_id = ?", "CONFIRMED", operatorID).
		Select("COALESCE(SUM(bookings.base_fare), 0)").
		Scan(&totalBaseFare).Error
	if err != nil {
		return nil, err
	}

	// 2. Refunds (from CANCELLED bookings)
	var totalRefunds float64
	err = r.db.Model(&model.Cancellation{}).
		Joins("JOIN bookings ON bookings.id = cancellations.booking_id").
		Joins("JOIN bus_instances ON bus_instances.id = bookings.bus_instance_id").
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("buses.operator_id = ? AND cancellations.refund_status = ?", operatorID, "SUCCESS").
		Select("COALESCE(SUM(cancellations.refund_amount), 0)").
		Scan(&totalRefunds).Error
	if err != nil {
		return nil, err
	}

	// 3. Tickets Sold count (Total from all bookings for this operator's buses)
	err = r.db.Model(&model.Passenger{}).
		Joins("JOIN bookings ON bookings.id = passengers.booking_id").
		Joins("JOIN bus_instances ON bus_instances.id = bookings.bus_instance_id").
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("bookings.status = ? AND buses.operator_id = ?", "CONFIRMED", operatorID).
		Count(&totalTicketsSold).Error
	if err != nil {
		return nil, err
	}

	// 4. Commission & Net Calculations
	var op model.Operator
	if err := r.db.Where("id = ?", operatorID).First(&op).Error; err != nil {
		return nil, err
	}

	// Commission is usually on the base fare
	commissionAmount := totalBaseFare * (op.CommissionRate / 100)
	netRevenue := totalBaseFare - commissionAmount - totalRefunds

	// 5. Top Performing Instances (with Bus Numbers)
	type InstanceStat struct {
		BusInstanceID string `json:"bus_instance_id"`
		BusNumber     string `json:"bus_number"`
		Count         int64  `json:"count"`
	}
	var instanceStats []InstanceStat
	r.db.Model(&model.Booking{}).
		Joins("JOIN bus_instances ON bus_instances.id = bookings.bus_instance_id").
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("bookings.status = ? AND buses.operator_id = ?", "CONFIRMED", operatorID).
		Group("bookings.bus_instance_id, buses.bus_number").
		Select("bookings.bus_instance_id, buses.bus_number, COUNT(*) as count").
		Order("count DESC").
		Limit(5).
		Scan(&instanceStats)

	return map[string]interface{}{
		"gross_amount":       grossAmount,
		"total_base_fare":    totalBaseFare,
		"net_revenue":        netRevenue,
		"commission_paid":    commissionAmount,
		"total_refunds":      totalRefunds,
		"total_bookings":     totalBookings,
		"total_tickets_sold": totalTicketsSold,
		"top_instances":      instanceStats,
		"commission_rate":    op.CommissionRate,
	}, nil
}

func (r *operatorRepository) WithTransaction(fn func(repo OperatorRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		return fn(&operatorRepository{db: tx})
	})
}

func (r *operatorRepository) FindOperatorByID(id string) (*model.Operator, error) {
	var op model.Operator
	err := r.db.Where("id = ?", id).First(&op).Error
	return &op, err
}

func (r *operatorRepository) GetInstancesByOperator(opID string) ([]model.BusInstance, error) {
	var instances []model.BusInstance
	err := r.db.
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("buses.operator_id = ?", opID).
		Preload("Bus.OriginStop").
		Preload("Bus.DestinationStop").
		Preload("Bus.BusType").
		Find(&instances).Error
	return instances, err
}

func (r *operatorRepository) DeleteBusInstance(opID, instanceID string) error {
	id, err := uuid.Parse(instanceID)
	if err != nil {
		return errors.New("invalid instance id format")
	}

	// Security check: verify ownership
	var count int64
	err = r.db.Model(&model.BusInstance{}).
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("bus_instances.id = ? AND buses.operator_id = ?", instanceID, opID).
		Count(&count).Error

	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("forbidden: instance not found or does not belong to your fleet")
	}

	// Mirror Admin Logic with added Safety Check
	return r.db.Transaction(func(tx *gorm.DB) error {
		// SAFETY: Check for confirmed bookings
		var bookingCount int64
		if err := tx.Model(&model.Booking{}).Where("bus_instance_id = ? AND status = ?", id, "CONFIRMED").Count(&bookingCount).Error; err != nil {
			return err
		}
		if bookingCount > 0 {
			return errors.New("cannot delete instance: confirmed bookings exist. Please cancel bookings first")
		}

		// 1. Delete associated records first
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
		return tx.Delete(&model.BusInstance{}, id).Error
	})
}

func (r *operatorRepository) UpdateInstanceStatus(opID, instanceID, status string) error {
	// Security check: verify ownership
	var count int64
	err := r.db.Model(&model.BusInstance{}).
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("bus_instances.id = ? AND buses.operator_id = ?", instanceID, opID).
		Count(&count).Error

	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("forbidden: instance not found or does not belong to your fleet")
	}

	return r.db.Model(&model.BusInstance{}).
		Where("id = ?", instanceID).
		Update("status", status).Error
}
