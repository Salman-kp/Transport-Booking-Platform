package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/nabeel-mp/tripneo/train-service/db"
	domainerrors "github.com/nabeel-mp/tripneo/train-service/domain_errors"
	"github.com/nabeel-mp/tripneo/train-service/kafka"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"github.com/nabeel-mp/tripneo/train-service/repository"
	"github.com/nabeel-mp/tripneo/train-service/utils"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type BookingRequest struct {
	TrainScheduleID string             `json:"train_schedule_id" validate:"required,uuid"`
	FromStationCode string             `json:"from_station_code"  validate:"required"`
	ToStationCode   string             `json:"to_station_code"    validate:"required"`
	Class           string             `json:"class"              validate:"required,oneof=SL 3AC 2AC 1AC"`
	SeatIDs         []string           `json:"seat_ids"           validate:"required,min=1"`
	Passengers      []PassengerRequest `json:"passengers"         validate:"required,min=1"`
}

type PassengerRequest struct {
	FirstName      string `json:"first_name"      validate:"required"`
	LastName       string `json:"last_name"       validate:"required"`
	DOB            string `json:"dob"             validate:"required"` // YYYY-MM-DD
	Gender         string `json:"gender"          validate:"required,oneof=male female other"`
	PassengerType  string `json:"passenger_type"  validate:"required,oneof=adult child infant"`
	IDType         string `json:"id_type"         validate:"required,oneof=AADHAAR PAN PASSPORT"`
	IDNumber       string `json:"id_number"       validate:"required"`
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
	// PaymentURL would be populated by the payment service; left empty until integrated.
	PaymentURL string `json:"payment_url"`
}

// resolveStopTime returns the absolute departure/arrival datetime for a given
// station code within a schedule, based on the train's stop data + DayOffset.
func resolveStopTime(schedule *models.TrainSchedule, stationCode string, departure bool) time.Time {
	baseDate := schedule.ScheduleDate
	for _, stop := range schedule.Train.Stops {
		if stop.Station.Code != stationCode {
			continue
		}
		timeStr := stop.ArrivalTime
		if departure {
			timeStr = stop.DepartureTime
		}
		var h, m int
		fmt.Sscanf(timeStr, "%d:%d", &h, &m)
		offsetDate := baseDate.AddDate(0, 0, stop.DayOffset)
		return time.Date(offsetDate.Year(), offsetDate.Month(), offsetDate.Day(),
			h, m, 0, 0, time.Local)
	}
	// Fallback to schedule-level times if stop not found
	if departure {
		return schedule.DepartureAt
	}
	return schedule.ArrivalAt
}

func CreateBooking(
	ctx context.Context,
	rdb *goredis.Client,
	userID string,
	req BookingRequest,
	producer *kafka.Producer,
) (*BookingResponse, error) {

	// Step 1 — Validate schedule exists and preload stops + stations
	schedule, err := repository.GetScheduleByID(req.TrainScheduleID)
	if err != nil {
		return nil, err
	}

	// Step 2 — Resolve boarding/alighting stations from the schedule's stops
	var fromStation, toStation models.Station
	var fromStop, toStop *models.TrainStop
	for i := range schedule.Train.Stops {
		s := &schedule.Train.Stops[i]
		if s.Station.Code == req.FromStationCode {
			fromStation = s.Station
			fromStop = s
		}
		if s.Station.Code == req.ToStationCode {
			toStation = s.Station
			toStop = s
		}
	}
	if fromStop == nil {
		return nil, fmt.Errorf("station %s is not a stop on this train", req.FromStationCode)
	}
	if toStop == nil {
		return nil, fmt.Errorf("station %s is not a stop on this train", req.ToStationCode)
	}
	if fromStop.StopSequence >= toStop.StopSequence {
		return nil, fmt.Errorf("from_station must come before to_station on this route")
	}

	departureTime := resolveStopTime(schedule, req.FromStationCode, true)
	arrivalTime := resolveStopTime(schedule, req.ToStationCode, false)

	// Step 3 — Validate all requested seats exist and are AVAILABLE
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

	// Step 4 — Lock all seats in Redis (all-or-nothing)
	lockErr, conflictSeatID := utils.LockSeats(ctx, rdb, req.TrainScheduleID, req.SeatIDs, userID)
	if lockErr != nil {
		return nil, fmt.Errorf("seat lock redis error: %w", lockErr)
	}
	if conflictSeatID != "" {
		return nil, domainerrors.ErrSeatAlreadyLocked
	}

	// Step 5 — Calculate total price
	var totalFare float64
	for _, seat := range seats {
		totalFare += seat.Price
	}
	serviceFee := math.Round(totalFare*0.02*100) / 100 // 2% service fee
	totalAmount := totalFare + serviceFee

	// Step 6 — Generate PNR
	pnr, err := utils.GeneratePNR()
	if err != nil {
		_ = utils.UnlockSeats(ctx, rdb, req.TrainScheduleID, req.SeatIDs)
		return nil, fmt.Errorf("PNR generation failed: %w", err)
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	var booking models.TrainBooking

	// Step 7 — DB transaction
	txErr := db.DB.Transaction(func(tx *gorm.DB) error {

		booking = models.TrainBooking{
			PNR:           pnr,
			UserID:        userID,
			ScheduleID:    uuid.MustParse(req.TrainScheduleID),
			FromStationID: fromStation.ID,
			ToStationID:   toStation.ID,
			DepartureTime: departureTime,
			ArrivalTime:   arrivalTime,
			SeatClass:     req.Class,
			Status:        "PENDING_PAYMENT",
			BaseFare:      totalFare,
			Taxes:         0,
			ServiceFee:    serviceFee,
			TotalAmount:   totalAmount,
			Currency:      "INR",
			BookedAt:      time.Now(),
			ExpiresAt:     &expiresAt,
		}
		if err := repository.CreateBooking(tx, &booking); err != nil {
			return err
		}

		if err := repository.UpdateSeatStatuses(tx, req.SeatIDs, "PENDING_PAYMENT"); err != nil {
			return err
		}

		if err := repository.DecrementAvailability(tx, req.TrainScheduleID, req.Class, len(req.SeatIDs)); err != nil {
			return err
		}

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
		_ = utils.UnlockSeats(ctx, rdb, req.TrainScheduleID, req.SeatIDs)
		return nil, txErr
	}

	// Step 8 — Publish BookingCreated event (payment service listens to this)
	producer.PublishBookingCreated(ctx, kafka.BookingCreatedEvent{
		BookingID:      booking.ID.String(),
		PNR:            booking.PNR,
		UserID:         booking.UserID,
		TrainName:      schedule.Train.TrainName,
		TrainNumber:    schedule.Train.TrainNumber,
		From:           req.FromStationCode,
		To:             req.ToStationCode,
		Departure:      departureTime.Format(time.RFC3339),
		Class:          req.Class,
		TotalAmount:    booking.TotalAmount,
		PassengerCount: len(req.Passengers),
	})

	return &BookingResponse{
		BookingID:   booking.ID.String(),
		PNR:         booking.PNR,
		Status:      booking.Status,
		TotalAmount: booking.TotalAmount,
		ExpiresAt:   expiresAt,
		PaymentURL:  "", // populated by payment service after it consumes train.booking.created
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
	producer *kafka.Producer,
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

	hoursLeft := int(time.Until(booking.TrainSchedule.DepartureAt).Hours())

	policy, err := repository.GetActiveCancellationPolicy(hoursLeft)
	if err != nil {
		return nil, err
	}

	refundAmount := booking.TotalAmount * (policy.RefundPercentage / 100)

	seatIDs, err := repository.GetSeatIDsByBooking(bookingID)
	if err != nil {
		return nil, err
	}

	var cancellation models.Cancellation
	txErr := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := repository.CancelBooking(tx, bookingID); err != nil {
			return err
		}
		if err := repository.UpdateSeatStatuses(tx, seatIDs, "AVAILABLE"); err != nil {
			return err
		}
		if err := repository.IncrementAvailability(
			tx, booking.ScheduleID.String(), booking.SeatClass, len(seatIDs),
		); err != nil {
			return err
		}
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

	_ = utils.UnlockSeats(ctx, rdb, booking.ScheduleID.String(), seatIDs)

	// Notify payment service to process refund (if booking was CONFIRMED)
	if booking.Status == "CONFIRMED" {
		producer.PublishBookingCancelled(ctx, kafka.BookingCancelledEvent{
			BookingID:    booking.ID.String(),
			PNR:          booking.PNR,
			UserID:       booking.UserID,
			RefundAmount: refundAmount,
			Reason:       "user_requested",
		})
	}

	return &cancellation, nil
}
