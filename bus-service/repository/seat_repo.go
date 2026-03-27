package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

// Fetch exact seat metadata checking for conflicts
func FindSeatByBusInstanceAndNumber(busInstanceID, seatNumber string) (*model.Seat, error) {
	var seat model.Seat
	err := db.DB.Where("bus_instance_id = ? AND seat_number = ?", busInstanceID, seatNumber).First(&seat).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &seat, err
}

// Toggle a single seat's availability safely
func UpdateSeatAvailability(seatID string, isAvailable bool) error {
	return db.DB.Model(&model.Seat{}).Where("id = ?", seatID).Update("is_available", isAvailable).Error
}

// Toggle an array of seats (mass book / mass expire logic)
func UpdateMultipleSeatsAvailability(seatIDs []string, isAvailable bool) error {
	if len(seatIDs) == 0 {
		return nil
	}
	return db.DB.Model(&model.Seat{}).Where("id IN ?", seatIDs).Update("is_available", isAvailable).Error
}
