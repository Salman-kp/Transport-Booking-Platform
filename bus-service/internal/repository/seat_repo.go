package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/internal/model"
	"gorm.io/gorm"
)

// Get all active seats for a bus
func FindSeatsByBusID(busID string) ([]model.Seat, error) {
	var seats []model.Seat
	err := db.DB.Where("bus_id = ? AND is_active = true", busID).Find(&seats).Error
	return seats, err
}

// Get seat by ID
func FindSeatByID(id string) (*model.Seat, error) {
	var seat model.Seat
	err := db.DB.Where("id = ?", id).First(&seat).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &seat, err
}

// Create new seat
func CreateSeat(seat *model.Seat) error {
	return db.DB.Create(seat).Error
}
