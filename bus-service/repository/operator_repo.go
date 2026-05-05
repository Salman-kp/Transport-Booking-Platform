package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

type OperatorRepository interface {
	CreateOperator(op *model.Operator) error
	CreateOperatorUser(opUser *model.OperatorUser) error
	FindOperatorByID(id string) (*model.Operator, error)
	GetOperatorUserByUserID(userID string) (*model.OperatorUser, error)
	UpdateOperatorStatus(id, status string) error
	LoadInventory(inv *model.OperatorInventory) error
	GetInventoryByOperator(operatorID string) ([]model.OperatorInventory, error)
	IncrementInventorySold(inventoryID string, quantity int) error
	GetBookingsByInventory(inventoryID, operatorID string) ([]model.Booking, error)
	VerifyBusInstanceOwnership(operatorID, busInstanceID string) (bool, error)
	GetOperatorAnalytics(operatorID string) (map[string]interface{}, error)
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

func (r *operatorRepository) FindOperatorByID(id string) (*model.Operator, error) {
	var op model.Operator
	err := r.db.Where("id = ?", id).First(&op).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &op, err
}

func (r *operatorRepository) GetOperatorUserByUserID(userID string) (*model.OperatorUser, error) {
	var opUser model.OperatorUser
	err := r.db.Preload("Operator").Where("user_id = ?", userID).First(&opUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &opUser, err
}

func (r *operatorRepository) UpdateOperatorStatus(id, status string) error {
	return r.db.Model(&model.Operator{}).Where("id = ?", id).Update("status", status).Error
}

func (r *operatorRepository) VerifyBusInstanceOwnership(operatorID, busInstanceID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.BusInstance{}).
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Where("bus_instances.id = ? AND buses.operator_id = ?", busInstanceID, operatorID).
		Count(&count).Error
	return count > 0, err
}

func (r *operatorRepository) LoadInventory(inv *model.OperatorInventory) error {
	return r.db.Create(inv).Error
}

func (r *operatorRepository) GetInventoryByOperator(operatorID string) ([]model.OperatorInventory, error) {
	var inventories []model.OperatorInventory
	err := r.db.Where("operator_id = ?", operatorID).Find(&inventories).Error
	return inventories, err
}

func (r *operatorRepository) IncrementInventorySold(inventoryID string, quantity int) error {
	currQuery := r.db.Model(&model.OperatorInventory{}).
		Where("id = ? AND quantity_loaded >= (quantity_sold + ?)", inventoryID, quantity)

	result := currQuery.UpdateColumn("quantity_sold", gorm.Expr("quantity_sold + ?", quantity))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("insufficient operator inventory available")
	}
	return nil
}

func (r *operatorRepository) GetBookingsByInventory(inventoryID, operatorID string) ([]model.Booking, error) {
	var bookings []model.Booking
	// Ensure that the inventory belongs to this operator
	err := r.db.
		Joins("JOIN operator_inventories ON operator_inventories.id = bookings.operator_inventory_id").
		Where("bookings.operator_inventory_id = ? AND operator_inventories.operator_id = ?", inventoryID, operatorID).
		Preload("Passengers").
		Find(&bookings).Error
	return bookings, err
}

func (r *operatorRepository) GetOperatorAnalytics(operatorID string) (map[string]interface{}, error) {
	var totalRevenue float64
	var totalBookings int64
	var totalTicketsSold int64

	// Count bookings and total revenue based on OperatorInventory links
	err := r.db.Model(&model.Booking{}).
		Joins("JOIN operator_inventories ON operator_inventories.id = bookings.operator_inventory_id").
		Where("bookings.status = ? AND operator_inventories.operator_id = ?", "CONFIRMED", operatorID).
		Count(&totalBookings).Error
	if err != nil {
		return nil, err
	}

	err = r.db.Model(&model.Booking{}).
		Joins("JOIN operator_inventories ON operator_inventories.id = bookings.operator_inventory_id").
		Where("bookings.status = ? AND operator_inventories.operator_id = ?", "CONFIRMED", operatorID).
		Select("COALESCE(SUM(bookings.total_amount), 0)").
		Scan(&totalRevenue).Error
	if err != nil {
		return nil, err
	}

	// Calculate total tickets sold (from inventory blocks)
	err = r.db.Model(&model.OperatorInventory{}).
		Where("operator_id = ?", operatorID).
		Select("COALESCE(SUM(quantity_sold), 0)").
		Scan(&totalTicketsSold).Error
	if err != nil {
		return nil, err
	}

	// Fetch commission rate for this operator
	var op model.Operator
	if err := r.db.Where("id = ?", operatorID).First(&op).Error; err != nil {
		return nil, err
	}
	
	// The operator takes the total revenue minus the admin's commission rate
	commissionAmount := totalRevenue * (op.CommissionRate / 100)
	netRevenue := totalRevenue - commissionAmount

	return map[string]interface{}{
		"total_revenue":      totalRevenue,
		"net_revenue":        netRevenue,
		"commission_paid":    commissionAmount,
		"total_bookings":     totalBookings,
		"total_tickets_sold": totalTicketsSold,
	}, nil
}
