package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

// Create booking with passengers (transaction-safe)
func CreateBooking(booking *model.Booking, passengers []model.Passenger) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(booking).Error; err != nil {
			return err
		}

		for i := range passengers {
			passengers[i].BookingID = booking.ID
		}

		if err := tx.Create(&passengers).Error; err != nil {
			return err
		}

		return nil
	})
}

// Get booking by ID with full relations
func FindBookingByID(id string) (*model.Booking, error) {
	var booking model.Booking
	err := db.DB.Preload("BusInstance.Bus").Preload("FareType").
		Preload("BoardingPoint.BusStop").Preload("DroppingPoint.BusStop").
		Where("id = ?", id).First(&booking).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &booking, err
}

func FindBookingsByUserID(userID string) ([]model.Booking, error) {
	var bookings []model.Booking
	err := db.DB.Preload("BusInstance.Bus").Where("user_id = ?", userID).
		Order("created_at DESC").Find(&bookings).Error
	return bookings, err
}

// Update booking status
func UpdateBookingStatus(id, status, paymentRef string) error {
	updates := map[string]any{
		"status": status,
	}

	if paymentRef != "" {
		updates["payment_ref"] = paymentRef
	}

	return db.DB.Model(&model.Booking{}).Where("id = ?", id).Updates(updates).Error
}

// Find Booking by its PNR
func FindBookingByPNR(pnr string) (*model.Booking, error) {
	var booking model.Booking
	err := db.DB.Preload("BusInstance.Bus").Preload("FareType").
		Preload("BoardingPoint.BusStop").Preload("DroppingPoint.BusStop").
		Where("pnr = ?", pnr).First(&booking).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &booking, err
}

// Get the attached e_ticket for frontend QR displays 
func GetETicketByBookingID(bookingID string) (*model.ETicket, error) {
	var ticket model.ETicket
	err := db.DB.Where("booking_id = ?", bookingID).First(&ticket).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &ticket, err
}

// Safely generate and save e_ticket to PG after QR-creation API calls return
func SaveETicket(ticket *model.ETicket) error {
	return db.DB.Create(ticket).Error
}

// Log cancellation event mapping internally back to standard policies
func CreateCancellation(cancel *model.Cancellation) error {
	return db.DB.Create(cancel).Error
}
