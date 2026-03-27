package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

// Create an operator
func CreateOperator(op *model.Operator) error {
	return db.DB.Create(op).Error
}

// Find operator by ID
func FindOperatorByID(id string) (*model.Operator, error) {
	var op model.Operator
	err := db.DB.Where("id = ?", id).First(&op).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &op, err
}

// Update Operator status
func UpdateOperatorStatus(id, status string) error {
	return db.DB.Model(&model.Operator{}).Where("id = ?", id).Update("status", status).Error
}

// Load operator inventory allocations
func LoadInventory(inv *model.OperatorInventory) error {
	return db.DB.Create(inv).Error
}

// Get operator inventory blocks
func GetInventoryByOperator(operatorID string) ([]model.OperatorInventory, error) {
	var inventories []model.OperatorInventory
	err := db.DB.Preload("Operator").Where("operator_id = ?", operatorID).Find(&inventories).Error
	return inventories, err
}

// Thread-safe inventory reservation hook
func IncrementInventorySold(inventoryID string, quantity int) error {
	// Uses GORM expressions to increment sold, and natively block if quantity_sold > quantity_loaded
	currQuery := db.DB.Model(&model.OperatorInventory{}).
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
