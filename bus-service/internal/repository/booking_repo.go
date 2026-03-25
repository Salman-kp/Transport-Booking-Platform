package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/internal/model"
	"gorm.io/gorm"
)

// Create booking with seats (transaction-safe)
func CreateBooking(booking *model.Booking, seats []model.BookingSeat) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(booking).Error; err != nil {
			return err
		}

		for i := range seats {
			seats[i].BookingID = booking.ID
		}

		if err := tx.Create(&seats).Error; err != nil {
			return err
		}

		return nil
	})
}

// Get booking by ID with full relations
func FindBookingByID(id string) (*model.Booking, error) {
	var booking model.Booking
	err := db.DB.Preload("BookingSeats.Seat").Preload("Schedule.Bus").
		Preload("Schedule.Route").Where("id = ?", id).First(&booking).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &booking, err
}

func FindBookingsByUserID(userID string) ([]model.Booking, error) {
	var bookings []model.Booking
	err := db.DB.Preload("Schedule.Bus").Preload("Schedule.Route").Where("user_id = ?", userID).
		Order("created_at DESC").Find(&bookings).Error
	return bookings, err
}

// Update booking + payment status
func UpdateBookingStatus(id, status, paymentStatus, paymentRefID string) error {
	updates := map[string]any{
		"status":         status,
		"payment_status": paymentStatus,
	}

	if paymentRefID != "" {
		updates["payment_ref_id"] = paymentRefID
	}

	return db.DB.Model(&model.Booking{}).Where("id = ?", id).Updates(updates).Error
}
