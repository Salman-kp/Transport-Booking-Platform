package repository

import (
	"fmt"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/db"
	domainerrors "github.com/nabeel-mp/tripneo/train-service/domain_errors"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/gorm"
)

func CreateBooking(tx *gorm.DB, booking *models.TrainBooking) error {
	if err := tx.Create(booking).Error; err != nil {
		return fmt.Errorf("create booking failed: %w", err)
	}
	return nil
}

func CreateBookingSeats(tx *gorm.DB, seats []models.BookingSeat) error {
	if err := tx.Create(&seats).Error; err != nil {
		return fmt.Errorf("create booking seats failed: %w", err)
	}
	return nil
}

// GetBookingByID now preloads the specific boarding and destination stations
func GetBookingByID(bookingID string) (*models.TrainBooking, error) {
	var booking models.TrainBooking
	err := db.DB.
		Preload("TrainSchedule.Train.Stops.Station"). // Deep preload
		Preload("FromStation").
		Preload("ToStation").
		First(&booking, "id = ?", bookingID).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func GetBookingByPNR(pnr string) (*models.TrainBooking, error) {
	var booking models.TrainBooking
	err := db.DB.
		Preload("TrainSchedule.Train.Stops.Station").
		Preload("FromStation").
		Preload("ToStation").
		First(&booking, "pnr = ?", pnr).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrPNRNotFound
		}
		return nil, fmt.Errorf("db error: %w", err)
	}
	return &booking, nil
}

// GetBookingsByUserID updated to show boarding/destination in booking history
func GetBookingsByUserID(userID string) ([]models.TrainBooking, error) {
	var bookings []models.TrainBooking
	err := db.DB.
		Preload("TrainSchedule.Train").
		Preload("FromStation").
		Preload("ToStation").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&bookings).Error
	if err != nil {
		return nil, fmt.Errorf("db error: %w", err)
	}
	return bookings, nil
}

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

func CreateCancellation(tx *gorm.DB, c *models.Cancellation) error {
	if err := tx.Create(c).Error; err != nil {
		return fmt.Errorf("create cancellation failed: %w", err)
	}
	return nil
}

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

func CreateTicket(tx *gorm.DB, ticket *models.TrainTicket) error {
	if err := tx.Create(ticket).Error; err != nil {
		return fmt.Errorf("create ticket failed: %w", err)
	}
	return nil
}

// GetTicketByBookingID now ensures boarding station info is present for the ticket UI
func GetTicketByBookingID(bookingID string) (*models.TrainTicket, error) {
	var ticket models.TrainTicket
	err := db.DB.
		Preload("Booking.TrainSchedule.Train.Stops.Station").
		Preload("Booking.FromStation").
		Preload("Booking.ToStation").
		First(&ticket, "booking_id = ?", bookingID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrBookingNotConfirmed
		}
		return nil, fmt.Errorf("db error: %w", err)
	}
	return &ticket, nil
}

func UpdateSeatStatuses(tx *gorm.DB, seatIDs []string, status string) error {
	return tx.Table("train_seats").Where("id IN ?", seatIDs).Update("status", status).Error
}
