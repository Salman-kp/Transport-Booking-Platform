package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/internal/model"
	"gorm.io/gorm"
)

// Get single bus by ID
func FindBusByID(id string) (*model.Bus, error) {
	var bus model.Bus
	err := db.DB.Where("id = ?", id).First(&bus).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &bus, err
}

// Create new bus
func CreateBus(bus *model.Bus) error {
	return db.DB.Create(bus).Error
}

// Get all active buses
func GetAllActiveBuses() ([]model.Bus, error) {
	var buses []model.Bus
	err := db.DB.Where("status = ?", "active").Find(&buses).Error
	return buses, err
}
