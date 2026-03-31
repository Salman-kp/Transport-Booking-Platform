package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/nabeel-mp/tripneo/train-service/db"
	domainerrors "github.com/nabeel-mp/tripneo/train-service/domain_errors"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"github.com/nabeel-mp/tripneo/train-service/repository"
	"github.com/nabeel-mp/tripneo/train-service/utils"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type BookingRequest struct {
	TrainScheduleID string             `json:"train_schedule_id" validate:"required,uuid"`
	Class           string             `json:"class"             validate:"required,oneof=SL 3AC 2AC 1AC"`
	SeatIDs         []string           `json:"seat_ids"          validate:"required,min=1"`
	Passengers      []PassengerRequest `json:"passengers"        validate:"required,min=1"`
}

type PassengerRequest struct {
	FirstName      string `json:"first_name"       validate:"required"`
	LastName       string `json:"last_name"        validate:"required"`
	DOB            string `json:"dob"              validate:"required"` // YYYY-MM-DD
	Gender         string `json:"gender"           validate:"required,oneof=male female other"`
	PassengerType  string `json:"passenger_type"   validate:"required,oneof=adult child infant"`
	IDType         string `json:"id_type"          validate:"required,oneof=AADHAAR PAN PASSPORT"`
	IDNumber       string `json:"id_number"        validate:"required"`
	MealPreference string `json:"meal_preference"`
	IsPrimary      bool   `json:"is_primary"`
	SeatID         string `json:"seat_id"` // empty for infants
}

// BookingResponse is returned to the client after booking creation.
type BookingResponse struct {
	BookingID   string    `json:"booking_id"`
	PNR         string    `json:"pnr"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
	ExpiresAt   time.Time `json:"expires_at"`
	PaymentURL  string    `json:"payment_url"`
}

func CreateBooking(
	ctx context.Context,
	rdb *goredis.Client,
	userID string,
	req BookingRequest,
) (*BookingResponse, error) {

	// Step 1 — Validate schedule exists
	_, err := repository.GetScheduleByID(req.TrainScheduleID)
	if err != nil {
		return nil, err
	}

	// Step 2 — Validate all requested seats exist and are AVAILABLE
	seats, err := repository.GetSeatsByIDs(req.SeatIDs)
	if err != nil {
		return nil, err
	}
	if len(seats) != len(req.SeatIDs) {
		return nil, domainerrors.ErrNoSeatsAvailable
	}
	for _, seat := range seats {
		if seat.Status != "AVAILABLE" {
			return nil, domainerrors.ErrSeatAlreadyBooked
		}
		if seat.Class != req.Class {
			return nil, fmt.Errorf("seat %s is not in class %s", seat.ID, req.Class)
		}
	}

	// Step 3 — Lock all seats in Redis (all-or-nothing, atomic per seat)
	lockErr, conflictSeatID := utils.LockSeats(ctx, rdb, req.TrainScheduleID, req.SeatIDs, userID)
	if lockErr != nil {
		return nil, fmt.Errorf("seat lock redis error: %w", lockErr)
	}
	if conflictSeatID != "" {
		return nil, domainerrors.ErrSeatAlreadyLocked
	}

	// Step 4 — Calculate total price
	var totalFare float64
	for _, seat := range seats {
		totalFare += seat.Price
	}
	serviceFee := math.Round(totalFare*0.02*100) / 100 // 2% service fee
	totalAmount := totalFare + serviceFee

	// Step 5 — Generate PNR (retry on collision handled by DB unique constraint)
	pnr, err := utils.GeneratePNR()
	if err != nil {
		_ = utils.UnlockSeats(ctx, rdb, req.TrainScheduleID, req.SeatIDs)
		return nil, fmt.Errorf("PNR generation failed: %w", err)
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	var booking models.TrainBooking

	// Step 6 — DB transaction: create booking + update inventory + add passengers
	txErr := db.DB.Transaction(func(tx *gorm.DB) error {

		// 6a. Create booking record
		booking = models.TrainBooking{
			PNR:         pnr,
			UserID:      userID,
			ScheduleID:  uuid.MustParse(req.TrainScheduleID),
			SeatClass:   req.Class,
			Status:      "PENDING_PAYMENT",
			BaseFare:    totalFare,
			Taxes:       0,
			ServiceFee:  serviceFee,
			TotalAmount: totalAmount,
			Currency:    "INR",
			BookedAt:    time.Now(),
			ExpiresAt:   &expiresAt,
		}
		if err := repository.CreateBooking(tx, &booking); err != nil {
			return err
		}

		// 6b. Update Seat Statuses in DB to prevent concurrent reads picking them up
		if err := repository.UpdateSeatStatuses(tx, req.SeatIDs, "PENDING_PAYMENT"); err != nil {
			return err
		}

		// 6c. Decrement overall schedule availability
		if err := repository.DecrementAvailability(tx, req.TrainScheduleID, req.Class, len(req.SeatIDs)); err != nil {
			return err
		}

		// 6d. Create BookingSeat join records
		bookingSeats := make([]models.BookingSeat, len(req.SeatIDs))
		for i, seatID := range req.SeatIDs {
			seatUUID := uuid.MustParse(seatID)
			bookingSeats[i] = models.BookingSeat{
				BookingID: booking.ID,
				SeatID:    seatUUID,
			}
		}
		if err := repository.CreateBookingSeats(tx, bookingSeats); err != nil {
			return err
		}

		// 6e. Create Passenger records
		passengers := make([]models.Passenger, len(req.Passengers))
		for i, p := range req.Passengers {
			dob, parseErr := time.Parse("2006-01-02", p.DOB)
			if parseErr != nil {
				return fmt.Errorf("invalid DOB for passenger %d: %w", i, parseErr)
			}

			var seatIDPtr *uuid.UUID
			if p.SeatID != "" && p.PassengerType != "infant" {
				sUID := uuid.MustParse(p.SeatID)
				seatIDPtr = &sUID
			}
			passengers[i] = models.Passenger{
				BookingID:      booking.ID,
				SeatID:         seatIDPtr,
				FirstName:      p.FirstName,
				LastName:       p.LastName,
				DateOfBirth:    dob,
				Gender:         p.Gender,
				PassengerType:  p.PassengerType,
				IDType:         p.IDType,
				IDNumber:       p.IDNumber,
				MealPreference: p.MealPreference,
				IsPrimary:      p.IsPrimary,
			}
		}

		return repository.CreatePassengers(tx, passengers)
	})

	if txErr != nil {
		// Transaction failed — release all Redis locks so others can book
		_ = utils.UnlockSeats(ctx, rdb, req.TrainScheduleID, req.SeatIDs)
		return nil, txErr
	}

	return &BookingResponse{
		BookingID:   booking.ID.String(),
		PNR:         booking.PNR,
		Status:      booking.Status,
		TotalAmount: booking.TotalAmount,
		ExpiresAt:   expiresAt,
		PaymentURL:  "", // To be integrated with a payment gateway
	}, nil
}

// GetBooking returns a booking if it belongs to the requesting user.
func GetBooking(bookingID, userID string) (*models.TrainBooking, error) {
	booking, err := repository.GetBookingByID(bookingID)
	if err != nil {
		return nil, err
	}
	if booking.UserID != userID {
		return nil, domainerrors.ErrUnauthorized
	}
	return booking, nil
}

// GetUserBookingHistory returns all bookings for a user.
func GetUserBookingHistory(userID string) ([]models.TrainBooking, error) {
	return repository.GetBookingsByUserID(userID)
}

func CancelBookingByUser(
	ctx context.Context,
	rdb *goredis.Client,
	bookingID, userID string,
) (*models.Cancellation, error) {

	booking, err := repository.GetBookingByID(bookingID)
	if err != nil {
		return nil, err
	}
	if booking.UserID != userID {
		return nil, domainerrors.ErrUnauthorized
	}
	if booking.Status != "PENDING_PAYMENT" && booking.Status != "CONFIRMED" {
		return nil, domainerrors.ErrCannotCancel
	}

	// Calculate hours until departure
	hoursLeft := int(time.Until(booking.TrainSchedule.DepartureAt).Hours())

	// Get applicable refund policy
	policy, err := repository.GetActiveCancellationPolicy(hoursLeft)
	if err != nil {
		return nil, err
	}

	refundAmount := booking.TotalAmount * (policy.RefundPercentage / 100)

	// Get seat IDs for this booking
	seatIDs, err := repository.GetSeatIDsByBooking(bookingID)
	if err != nil {
		return nil, err
	}

	var cancellation models.Cancellation
	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Cancel booking
		if err := repository.CancelBooking(tx, bookingID); err != nil {
			return err
		}

		// 2. Restore seats to AVAILABLE
		if err := repository.UpdateSeatStatuses(tx, seatIDs, "AVAILABLE"); err != nil {
			return err
		}

		// 3. Restore availability count on schedule
		if err := repository.IncrementAvailability(
			tx,
			booking.ScheduleID.String(),
			booking.SeatClass,
			len(seatIDs),
		); err != nil {
			return err
		}

		// 4. Write cancellation record
		policyID := policy.ID
		cancellation = models.Cancellation{
			BookingID:       booking.ID,
			RefundAmount:    refundAmount,
			RefundStatus:    "PENDING",
			PolicyAppliedID: &policyID,
			RequestedAt:     time.Now(),
		}
		return repository.CreateCancellation(tx, &cancellation)
	})

	if txErr != nil {
		return nil, txErr
	}

	// Release Redis seat locks (already expired if booking was PENDING_PAYMENT, but safe to call)
	_ = utils.UnlockSeats(ctx, rdb, booking.ScheduleID.String(), seatIDs)

	return &cancellation, nil
}
