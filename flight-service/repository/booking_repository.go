package repository

import (
	"errors"
	"time"

	"github.com/junaid9001/tripneo/flight-service/models"
	"gorm.io/gorm"
)

type BookingRepository struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) CreateBooking(booking *models.Booking) error {
	return r.db.Create(booking).Error
}

func (r *BookingRepository) GetBookingByID(id string) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.Preload("Passengers").Preload("Ancillaries").Preload("FlightInstance.Flight.OriginAirport").Preload("FlightInstance.Flight.DestinationAirport").Preload("FareType").First(&booking, "id = ?", id).Error
	return &booking, err
}

func (r *BookingRepository) GetBookingByPNR(pnr string) (*models.Booking, error) {
	var booking models.Booking
	err := r.db.Preload("Passengers").Preload("Ancillaries").Preload("FlightInstance.Flight.OriginAirport").Preload("FlightInstance.Flight.DestinationAirport").Preload("FareType").First(&booking, "pnr = ?", pnr).Error
	return &booking, err
}

func (r *BookingRepository) GetBookingsByUserID(userID string) ([]models.Booking, error) {
	var bookings []models.Booking
	err := r.db.Preload("Passengers").Preload("FlightInstance.Flight.OriginAirport").Preload("FlightInstance.Flight.DestinationAirport").Where("user_id = ?", userID).Find(&bookings).Error
	return bookings, err
}

func (r *BookingRepository) UpdateBooking(booking *models.Booking) error {
	return r.db.Save(booking).Error
}

func (r *BookingRepository) CreateCancellation(cancellation *models.Cancellation, booking *models.Booking) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(cancellation).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Booking{}).Where("id = ?", cancellation.BookingID).Update("status", "CANCELLED").Error; err != nil {
			return err
		}

		for _, p := range booking.Passengers {
			if p.SeatID != nil {
				if err := tx.Model(&models.Seat{}).Where("id = ?", p.SeatID).Update("is_available", true).Error; err != nil {
					return err
				}
			}
		}

		// update availability counts
		if len(booking.Passengers) > 0 {
			col := "available_economy"
			quotaCol := "platform_quota_economy"
			if booking.SeatClass == "BUSINESS" {
				col = "available_business"
				quotaCol = "platform_quota_business"
			}
			if err := tx.Model(&models.FlightInstance{}).Where("id = ?", booking.FlightInstanceID).Updates(map[string]interface{}{
				col:      gorm.Expr(col+" + ?", len(booking.Passengers)),
				quotaCol: gorm.Expr(quotaCol+" + ?", len(booking.Passengers)),
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *BookingRepository) ConfirmBookingAndSeats(booking *models.Booking, ticket *models.ETicket) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// update booking status
		if err := tx.Save(booking).Error; err != nil {
			return err
		}

		for _, p := range booking.Passengers {
			if p.SeatID != nil {
				if err := tx.Model(&models.Seat{}).Where("id = ?", p.SeatID).Update("is_available", false).Error; err != nil {
					return err
				}
			}
		}

		if err := tx.Create(ticket).Error; err != nil {
			return err
		}

		// deduct availability counts
		if len(booking.Passengers) > 0 {
			col := "available_economy"
			quotaCol := "platform_quota_economy"
			if booking.SeatClass == "BUSINESS" {
				col = "available_business"
				quotaCol = "platform_quota_business"
			}
			if err := tx.Model(&models.FlightInstance{}).Where("id = ?", booking.FlightInstanceID).Updates(map[string]interface{}{
				col:      gorm.Expr(col+" - ?", len(booking.Passengers)),
				quotaCol: gorm.Expr(quotaCol+" - ?", len(booking.Passengers)),
			}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *BookingRepository) SaveETicket(ticket *models.ETicket) error {
	return r.db.Create(ticket).Error
}

func (r *BookingRepository) GetETicketByBookingID(bookingID string) (*models.ETicket, error) {
	var ticket models.ETicket
	err := r.db.First(&ticket, "booking_id = ?", bookingID).Error
	return &ticket, err
}

func (r *BookingRepository) GetFlightInstanceByID(id string) (*models.FlightInstance, error) {
	var instance models.FlightInstance
	err := r.db.First(&instance, "id = ?", id).Error
	return &instance, err
}

func (r *BookingRepository) GetFareTypeByID(id string) (*models.FareType, error) {
	var fare models.FareType
	err := r.db.First(&fare, "id = ?", id).Error
	return &fare, err
}

func (r *BookingRepository) GetCancellationByBookingID(bookingID string) (*models.Cancellation, error) {
	var cancellation models.Cancellation
	err := r.db.Where("booking_id = ?", bookingID).First(&cancellation).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cancellation, nil
}

func (r *BookingRepository) GetSeatByID(id string) (*models.Seat, error) {
	var seat models.Seat
	err := r.db.First(&seat, "id = ?", id).Error
	return &seat, err
}

func (r *BookingRepository) UpdateCancellationRefundStatus(bookingID string, status string) error {
	updates := map[string]interface{}{
		"refund_status": status,
	}
	if status == "COMPLETED" || status == "FAILED" {
		now := time.Now()
		updates["processed_at"] = &now
	}

	return r.db.Model(&models.Cancellation{}).
		Where("booking_id = ?", bookingID).
		Updates(updates).Error
}
