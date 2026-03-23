package repository

import (
	"fmt"

	"github.com/nabeel-mp/tripneo/train-service/db"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/gorm"
)

// CreatePassengers bulk-inserts passenger records for a booking.
func CreatePassengers(tx *gorm.DB, passengers []models.Passenger) error {
	if err := tx.Create(&passengers).Error; err != nil {
		return fmt.Errorf("create passengers failed: %w", err)
	}
	return nil
}

// GetPassengersByBookingID returns all passengers for a booking.
func GetPassengersByBookingID(bookingID string) ([]models.Passenger, error) {
	var passengers []models.Passenger
	err := db.DB.
		Where("booking_id = ?", bookingID).
		Find(&passengers).Error
	if err != nil {
		return nil, fmt.Errorf("db error fetching passengers: %w", err)
	}
	return passengers, nil
}
