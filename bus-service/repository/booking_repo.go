package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

type BookingRepository interface {
	// Core booking CRUD
	CreateBooking(booking *model.Booking, passengers []model.Passenger) error
	FindBookingByID(id string, userID string) (*model.Booking, error)
	FindBookingByPNR(pnr string, userID string) (*model.Booking, error)
	FindBookingsByUserID(userID string) ([]model.Booking, error)
	UpdateBookingStatus(id, userID, status, paymentRef string) error

	// Ticket
	SaveETicket(ticket *model.ETicket) error
	GetETicketByBookingID(bookingID string, userID string) (*model.ETicket, error)

	// Cancellation
	CreateCancellation(cancel *model.Cancellation) error
	GetActiveCancellationPolicy(hoursBeforeDeparture int) (*model.CancellationPolicy, error)

	// Validation helpers used during CreateBooking
	GetFareTypeByID(id string) (*model.FareType, error)
	GetSeatByID(id string) (*model.Seat, error)
	GetBoardingPointByID(id string) (*model.BoardingPoint, error)
	GetDroppingPointByID(id string) (*model.DroppingPoint, error)

	// Seat availability
	UpdateMultipleSeatsAvailability(seatIDs []string, isAvailable bool) error

	// Inventory — allocated model adjustments on confirm/cancel
	DecrementInventoryOnConfirm(busInstanceID, fareTypeID, seatType string, count int) error
	IncrementInventoryOnCancel(busInstanceID, fareTypeID, seatType string, count int) error
}

type bookingRepository struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository) CreateBooking(booking *model.Booking, passengers []model.Passenger) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
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

func (r *bookingRepository) FindBookingByID(id string, userID string) (*model.Booking, error) {
	var booking model.Booking
	err := r.db.
		Preload("BusInstance.Bus.Operator").
		Preload("BusInstance.Bus.OriginStop").
		Preload("BusInstance.Bus.DestinationStop").
		Preload("FareType").
		Preload("BoardingPoint.BusStop").
		Preload("DroppingPoint.BusStop").
		Preload("Passengers.Seat").
		Where("id = ? AND user_id = ?", id, userID).
		First(&booking).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("booking not found")
	}
	return &booking, err
}

func (r *bookingRepository) FindBookingByPNR(pnr string, userID string) (*model.Booking, error) {
	var booking model.Booking
	err := r.db.
		Preload("BusInstance.Bus.Operator").
		Preload("BusInstance.Bus.OriginStop").
		Preload("BusInstance.Bus.DestinationStop").
		Preload("FareType").
		Preload("BoardingPoint.BusStop").
		Preload("DroppingPoint.BusStop").
		Preload("Passengers.Seat").
		Where("pnr = ? AND user_id = ?", pnr, userID).
		First(&booking).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("booking not found")
	}
	return &booking, err
}

func (r *bookingRepository) FindBookingsByUserID(userID string) ([]model.Booking, error) {
	var bookings []model.Booking
	err := r.db.
		Preload("BusInstance.Bus.Operator").
		Preload("BusInstance.Bus.OriginStop").
		Preload("BusInstance.Bus.DestinationStop").
		Preload("Passengers").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&bookings).Error
	return bookings, err
}

func (r *bookingRepository) UpdateBookingStatus(id, userID, status, paymentRef string) error {
	updates := map[string]any{"status": status}
	if paymentRef != "" {
		updates["payment_ref"] = paymentRef
	}
	switch status {
	case "CONFIRMED":
		updates["confirmed_at"] = gorm.Expr("NOW()")
	case "CANCELLED":
		updates["cancelled_at"] = gorm.Expr("NOW()")
	}

	res := r.db.Model(&model.Booking{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("booking not found or access denied")
	}
	return nil
}

func (r *bookingRepository) SaveETicket(ticket *model.ETicket) error {
	return r.db.Create(ticket).Error
}

func (r *bookingRepository) GetETicketByBookingID(bookingID string, userID string) (*model.ETicket, error) {
	var ticket model.ETicket
	err := r.db.
		Joins("JOIN bookings ON bookings.id = e_tickets.booking_id").
		Where("e_tickets.booking_id = ? AND bookings.user_id = ?", bookingID, userID).
		Preload("Booking").
		First(&ticket).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("ticket not found")
	}
	return &ticket, err
}

func (r *bookingRepository) CreateCancellation(cancel *model.Cancellation) error {
	return r.db.Create(cancel).Error
}

// GetActiveCancellationPolicy returns the matching refund policy for the given
// hours remaining before departure. It picks the policy whose hours_before_departure
// is the smallest value that is still ≥ hoursLeft (i.e., the most specific bracket).
// Falls back to the 0-hour (no-refund) policy if none found.
func (r *bookingRepository) GetActiveCancellationPolicy(hoursLeft int) (*model.CancellationPolicy, error) {
	var policy model.CancellationPolicy
	err := r.db.
		Where("is_active = true AND hours_before_departure <= ?", hoursLeft).
		Order("hours_before_departure DESC").
		First(&policy).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// No policy covers this window — zero refund
		return &model.CancellationPolicy{
			Name:                 "No Refund",
			HoursBeforeDeparture: 0,
			RefundPercentage:     0,
		}, nil
	}
	return &policy, err
}

// ── Validation helpers ────────────────────────────────────────────────────────

func (r *bookingRepository) GetFareTypeByID(id string) (*model.FareType, error) {
	var fare model.FareType
	err := r.db.Where("id = ?", id).First(&fare).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("fare type not found")
	}
	return &fare, err
}

func (r *bookingRepository) GetSeatByID(id string) (*model.Seat, error) {
	var seat model.Seat
	err := r.db.Where("id = ?", id).First(&seat).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("seat not found")
	}
	return &seat, err
}

func (r *bookingRepository) GetBoardingPointByID(id string) (*model.BoardingPoint, error) {
	var bp model.BoardingPoint
	err := r.db.Preload("BusStop").Where("id = ?", id).First(&bp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("boarding point not found")
	}
	return &bp, err
}

func (r *bookingRepository) GetDroppingPointByID(id string) (*model.DroppingPoint, error) {
	var dp model.DroppingPoint
	err := r.db.Preload("BusStop").Where("id = ?", id).First(&dp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("dropping point not found")
	}
	return &dp, err
}

// ── Seat availability ─────────────────────────────────────────────────────────

func (r *bookingRepository) UpdateMultipleSeatsAvailability(seatIDs []string, isAvailable bool) error {
	if len(seatIDs) == 0 {
		return nil
	}
	return r.db.Model(&model.Seat{}).
		Where("id IN ?", seatIDs).
		Update("is_available", isAvailable).Error
}

// ── Inventory (allocated model) ───────────────────────────────────────────────

// DecrementInventoryOnConfirm decrements seats_remaining on the bus_instance and
// increments quantity_sold on operator_inventory when a booking is CONFIRMED.
func (r *bookingRepository) DecrementInventoryOnConfirm(busInstanceID, fareTypeID, seatType string, count int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Decrement the matching available_* column on bus_instances
		col := availableColumn(seatType)
		if col == "" {
			return nil // unknown seat type — skip
		}
		if err := tx.Model(&model.BusInstance{}).
			Where("id = ?", busInstanceID).
			UpdateColumn(col, gorm.Expr(col+" - ?", count)).Error; err != nil {
			return err
		}

		// Increment quantity_sold on operator_inventory
		if err := tx.Model(&model.OperatorInventory{}).
			Where("bus_instance_id = ? AND fare_type_id = ?", busInstanceID, fareTypeID).
			UpdateColumn("quantity_sold", gorm.Expr("quantity_sold + ?", count)).Error; err != nil {
			return err
		}
		return nil
	})
}

// IncrementInventoryOnCancel reverses the confirm operation when a booking is CANCELLED.
func (r *bookingRepository) IncrementInventoryOnCancel(busInstanceID, fareTypeID, seatType string, count int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		col := availableColumn(seatType)
		if col == "" {
			return nil
		}
		if err := tx.Model(&model.BusInstance{}).
			Where("id = ?", busInstanceID).
			UpdateColumn(col, gorm.Expr(col+" + ?", count)).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.OperatorInventory{}).
			Where("bus_instance_id = ? AND fare_type_id = ?", busInstanceID, fareTypeID).
			UpdateColumn("quantity_sold", gorm.Expr("quantity_sold - ?", count)).Error; err != nil {
			return err
		}
		return nil
	})
}

// availableColumn maps a seat type string to the corresponding column name.
func availableColumn(seatType string) string {
	switch seatType {
	case "SEATER":
		return "available_seater"
	case "SEMI_SLEEPER":
		return "available_semi_sleeper"
	case "SLEEPER":
		return "available_sleeper"
	default:
		return ""
	}
}
