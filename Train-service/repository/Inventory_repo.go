package repository

import (
	"fmt"

	"github.com/nabeel-mp/tripneo/train-service/db"
	domainerrors "github.com/nabeel-mp/tripneo/train-service/domain_errors"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/gorm"
)

// GetSeatsByScheduleAndClass returns all inventory seats for a schedule+class.
// Used to render the berth map.
func GetSeatsByScheduleAndClass(scheduleID, class string) ([]models.TrainInventory, error) {
	var seats []models.TrainInventory
	err := db.DB.
		Where("train_schedule_id = ? AND class = ?", scheduleID, class).
		Order("coach ASC, seat_number ASC").
		Find(&seats).Error
	if err != nil {
		return nil, fmt.Errorf("db error fetching seats: %w", err)
	}
	return seats, nil
}

// GetSeatByID fetches a single inventory seat.
func GetSeatByID(seatID string) (*models.TrainInventory, error) {
	var seat models.TrainInventory
	err := db.DB.First(&seat, "id = ?", seatID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrSeatAlreadyBooked
		}
		return nil, fmt.Errorf("db error: %w", err)
	}
	return &seat, nil
}

// GetSeatsByIDs fetches multiple inventory seats by their UUIDs.
// Used during booking to validate all requested seats exist and are AVAILABLE.
func GetSeatsByIDs(seatIDs []string) ([]models.TrainInventory, error) {
	var seats []models.TrainInventory
	err := db.DB.Where("id IN ?", seatIDs).Find(&seats).Error
	if err != nil {
		return nil, fmt.Errorf("db error fetching seats: %w", err)
	}
	return seats, nil
}

// MarkSeatsBooked updates seat status to BOOKED in a transaction.
// Called by the Kafka consumer after payment.completed event.
func MarkSeatsBooked(tx *gorm.DB, seatIDs []string) error {
	return tx.Model(&models.TrainInventory{}).
		Where("id IN ?", seatIDs).
		Update("status", "BOOKED").Error
}

// MarkSeatsAvailable restores seats to AVAILABLE in a transaction.
// Called on cancellation or expiry.
func MarkSeatsAvailable(tx *gorm.DB, seatIDs []string) error {
	return tx.Model(&models.TrainInventory{}).
		Where("id IN ?", seatIDs).
		Update("status", "AVAILABLE").Error
}

// GetSeatIDsByBooking returns all seat UUIDs linked to a booking.
// Used to build the unlock list for expiry and cancellation.
func GetSeatIDsByBooking(bookingID string) ([]string, error) {
	var ids []string
	err := db.DB.
		Model(&models.BookingSeat{}).
		Select("seat_id").
		Where("booking_id = ?", bookingID).
		Pluck("seat_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("db error fetching seat ids: %w", err)
	}
	return ids, nil
}
