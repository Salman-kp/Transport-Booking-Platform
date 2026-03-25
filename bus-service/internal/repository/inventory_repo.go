package repository

import (
	"errors"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Upsert inventory by schedule_id (atomic)
func UpsertInventory(inv *model.Inventory) error {
	return db.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "schedule_id"}},
		UpdateAll: true,
	}).Create(inv).Error
}

// Get inventory by schedule
func GetInventoryBySchedule(scheduleID string) (*model.Inventory, error) {
	var inv model.Inventory
	err := db.DB.Where("schedule_id = ?", scheduleID).
		First(&inv).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &inv, err
}

// Get inventory by schedule + date
func GetInventoryByScheduleAndDate(scheduleID string, date time.Time) (*model.Inventory, error) {
	var inv model.Inventory
	err := db.DB.
		Where("schedule_id = ? AND valid_date = ?", scheduleID, date).
		First(&inv).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &inv, err
}

// Get all inventory
func GetAllInventory() ([]model.Inventory, error) {
	var invs []model.Inventory
	err := db.DB.Find(&invs).Error
	return invs, err
}
