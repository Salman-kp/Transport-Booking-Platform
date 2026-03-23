package repository

import (
	"fmt"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/db"
	domainerrors "github.com/nabeel-mp/tripneo/train-service/domain_errors"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/gorm"
)

// CreateBooking inserts a new TrainBooking record.
func CreateBooking(tx *gorm.DB, booking *models.TrainBooking) error {
	if err := tx.Create(booking).Error; err != nil {
		return fmt.Errorf("create booking failed: %w", err)
	}
	return nil
}

// CreateBookingSeats bulk-inserts the seat join records.
func CreateBookingSeats(tx *gorm.DB, seats []models.BookingSeat) error {
	if err := tx.Create(&seats).Error; err != nil {
		return fmt.Errorf("create booking seats failed: %w", err)
	}
	return nil
}

// GetBookingByID fetches a booking with Schedule + Train preloaded.
func GetBookingByID(bookingID string) (*models.TrainBooking, error) {
	var booking models.TrainBooking
	err := db.DB.
		Preload("TrainSchedule.Train").
		First(&booking, "id = ?", bookingID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrBookingNotFound
		}
		return nil, fmt.Errorf("db error: %w", err)
	}
	return &booking, nil
}

// GetBookingByPNR fetches a booking by PNR string.
func GetBookingByPNR(pnr string) (*models.TrainBooking, error) {
	var booking models.TrainBooking
	err := db.DB.
		Preload("TrainSchedule.Train").
		First(&booking, "pnr = ?", pnr).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrPNRNotFound
		}
		return nil, fmt.Errorf("db error: %w", err)
	}
	return &booking, nil
}

// GetBookingsByUserID returns booking history for a user, newest first.
func GetBookingsByUserID(userID string) ([]models.TrainBooking, error) {
	var bookings []models.TrainBooking
	err := db.DB.
		Preload("TrainSchedule.Train").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&bookings).Error
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	return bookings, nil
}

// UpdateBookingStatus updates the booking status field.
func UpdateBookingStatus(tx *gorm.DB, bookingID, status string) error {
	result := tx.Model(&models.TrainBooking{}).
		Where("id = ?", bookingID).
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("update booking status failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrBookingNotFound
	}
	return nil
}

// ConfirmBooking sets status=CONFIRMED and confirmed_at timestamp.
func ConfirmBooking(tx *gorm.DB, bookingID, paymentRef string) error {
	now := time.Now()
	return tx.Model(&models.TrainBooking{}).
		Where("id = ?", bookingID).
		Updates(map[string]interface{}{
			"status":       "CONFIRMED",
			"confirmed_at": now,
			"payment_ref":  paymentRef,
			"updated_at":   now,
		}).Error
}

// CancelBooking sets status=CANCELLED and cancelled_at timestamp.
func CancelBooking(tx *gorm.DB, bookingID string) error {
	now := time.Now()
	return tx.Model(&models.TrainBooking{}).
		Where("id = ?", bookingID).
		Updates(map[string]interface{}{
			"status":       "CANCELLED",
			"cancelled_at": now,
			"updated_at":   now,
		}).Error
}

// GetExpiredPendingBookings returns all bookings that are
// PENDING_PAYMENT and past their expires_at timestamp.
// Called by the expiry background worker every 5 minutes.
func GetExpiredPendingBookings() ([]models.TrainBooking, error) {
	var bookings []models.TrainBooking
	err := db.DB.
		Where("status = ? AND expires_at < ?", "PENDING_PAYMENT", time.Now()).
		Find(&bookings).Error
	if err != nil {
		return nil, fmt.Errorf("db error fetching expired bookings: %w", err)
	}
	return bookings, nil
}

// CreateCancellation inserts a Cancellation record.
func CreateCancellation(tx *gorm.DB, c *models.Cancellation) error {
	if err := tx.Create(c).Error; err != nil {
		return fmt.Errorf("create cancellation failed: %w", err)
	}
	return nil
}

// GetActiveCancellationPolicy finds the best policy for a given
// hours-before-departure value. Returns the policy with the highest
// HoursBeforeDeparture that is <= hoursLeft.
func GetActiveCancellationPolicy(hoursLeft int) (*models.CancellationPolicy, error) {
	var policy models.CancellationPolicy
	err := db.DB.
		Where("is_active = true AND hours_before_departure <= ?", hoursLeft).
		Order("hours_before_departure DESC").
		First(&policy).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrRefundNotEligible
		}
		return nil, fmt.Errorf("db error fetching policy: %w", err)
	}
	return &policy, nil
}

// CreateTicket inserts a TrainTicket record after QR generation.
func CreateTicket(tx *gorm.DB, ticket *models.TrainTicket) error {
	if err := tx.Create(ticket).Error; err != nil {
		return fmt.Errorf("create ticket failed: %w", err)
	}
	return nil
}

// GetTicketByBookingID fetches the ticket for a confirmed booking.
func GetTicketByBookingID(bookingID string) (*models.TrainTicket, error) {
	var ticket models.TrainTicket
	err := db.DB.
		Preload("Booking.TrainSchedule.Train").
		First(&ticket, "booking_id = ?", bookingID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrBookingNotConfirmed
		}
		return nil, fmt.Errorf("db error: %w", err)
	}
	return &ticket, nil
}
