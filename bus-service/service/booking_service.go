package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/config"
	"github.com/Salman-kp/tripneo/bus-service/dto"
	"github.com/Salman-kp/tripneo/bus-service/model"
	busredis "github.com/Salman-kp/tripneo/bus-service/redis"
	"github.com/Salman-kp/tripneo/bus-service/redpanda"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// BookingService defines all booking lifecycle operations.
type BookingService interface {
	CreateBooking(userID string, req dto.CreateBookingRequest) (*dto.BookingResponse, error)
	GetBookingByID(id string, userID string) (*dto.BookingResponse, error)
	GetBookingByPNR(pnr string, userID string) (*dto.BookingResponse, error)
	GetUserBookings(userID string) ([]dto.BookingResponse, error)
	ConfirmBooking(id string, userID string) error
	CancelBooking(id string, userID string, req *dto.CancelBookingRequest) (*dto.CancelBookingResponse, error)
	GetBookingTicket(id string, userID string) (*dto.TicketResponse, error)
}

type bookingService struct {
	repo     repository.BookingRepository
	rdb      *goredis.Client
	producer *redpanda.Producer
}

// NewBookingService constructs a BookingService.
// producer may be nil in local dev (Redpanda disabled).
func NewBookingService(repo repository.BookingRepository, rdb *goredis.Client, producer *redpanda.Producer) BookingService {
	return &bookingService{repo: repo, rdb: rdb, producer: producer}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// generatePNR generates a cryptographically random 6-character uppercase alphanumeric PNR.
func generatePNR() string {
	b := make([]byte, 3)
	rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

// extractSeatIDs collects seat UUIDs from a passenger slice.
func extractSeatIDs(passengers []model.Passenger) []string {
	ids := make([]string, 0, len(passengers))
	for _, p := range passengers {
		if p.SeatID != nil {
			ids = append(ids, p.SeatID.String())
		}
	}
	return ids
}

// bookingToDTO converts a model.Booking to a dto.BookingResponse.
func bookingToDTO(b *model.Booking) *dto.BookingResponse {
	resp := &dto.BookingResponse{
		ID:            b.ID.String(),
		PNR:           b.PNR,
		Status:        b.Status,
		BusInstanceID: b.BusInstanceID.String(),
		SeatType:      b.SeatType,
		BaseFare:      b.BaseFare,
		Taxes:         b.Taxes,
		ServiceFee:    b.ServiceFee,
		TotalAmount:   b.TotalAmount,
		Currency:      b.Currency,
		BookedAt:      b.BookedAt,
		ExpiresAt:     b.ExpiresAt,
		PaymentRef:    b.PaymentRef,
		// PaymentURL is empty until the Payment Service is integrated.
		// The future Payment Service listens to bus.booking.created and generates the Stripe URL.
		PaymentURL: "",
	}

	if b.BoardingPoint.BusStop.Name != "" {
		resp.BoardingPoint = b.BoardingPoint.BusStop.Name
		if b.BoardingPoint.Landmark != "" {
			resp.BoardingPoint += " — " + b.BoardingPoint.Landmark
		}
	}
	if b.DroppingPoint.BusStop.Name != "" {
		resp.DroppingPoint = b.DroppingPoint.BusStop.Name
		if b.DroppingPoint.Landmark != "" {
			resp.DroppingPoint += " — " + b.DroppingPoint.Landmark
		}
	}

	for _, p := range b.Passengers {
		pd := dto.PassengerDetails{
			ID:            p.ID.String(),
			FirstName:     p.FirstName,
			LastName:      p.LastName,
			PassengerType: p.PassengerType,
		}
		if p.Seat != nil {
			pd.SeatNumber = p.Seat.SeatNumber
		}
		resp.Passengers = append(resp.Passengers, pd)
	}
	return resp
}

// ── CreateBooking ─────────────────────────────────────────────────────────────

func (s *bookingService) CreateBooking(userID string, req dto.CreateBookingRequest) (*dto.BookingResponse, error) {
	ctx := context.Background()

	// ── Parse UUIDs ──────────────────────────────────────────────────────────
	busInstanceID, err := uuid.Parse(req.BusInstanceID)
	if err != nil {
		return nil, errors.New("invalid bus_instance_id format")
	}
	fareTypeID, err := uuid.Parse(req.FareTypeID)
	if err != nil {
		return nil, errors.New("invalid fare_type_id format")
	}
	boardingID, err := uuid.Parse(req.BoardingPointID)
	if err != nil {
		return nil, errors.New("invalid boarding_point_id format")
	}
	droppingID, err := uuid.Parse(req.DroppingPointID)
	if err != nil {
		return nil, errors.New("invalid dropping_point_id format")
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}
	if len(req.Passengers) == 0 {
		return nil, errors.New("at least one passenger is required")
	}

	// ── Validate fare type belongs to this bus instance ───────────────────────
	fareType, err := s.repo.GetFareTypeByID(req.FareTypeID)
	if err != nil {
		return nil, errors.New("fare type not found")
	}
	if fareType.BusInstanceID != busInstanceID {
		return nil, errors.New("fare type does not belong to the selected bus")
	}

	// ── Validate boarding/dropping points ─────────────────────────────────────
	bp, err := s.repo.GetBoardingPointByID(req.BoardingPointID)
	if err != nil {
		return nil, errors.New("boarding point not found")
	}
	if bp.BusInstanceID != busInstanceID {
		return nil, errors.New("boarding point does not belong to the selected bus")
	}

	dp, err := s.repo.GetDroppingPointByID(req.DroppingPointID)
	if err != nil {
		return nil, errors.New("dropping point not found")
	}
	if dp.BusInstanceID != busInstanceID {
		return nil, errors.New("dropping point does not belong to the selected bus")
	}
	if bp.SequenceOrder >= dp.SequenceOrder {
		return nil, errors.New("dropping point must be after boarding point in the route sequence")
	}

	// ── Validate each seat and build passenger list ───────────────────────────
	seatIDs := make([]string, 0, len(req.Passengers))
	passengers := make([]model.Passenger, 0, len(req.Passengers))
	var baseFareTotal float64
	isPrimarySet := false

	for i, pReq := range req.Passengers {
		seatUUID, err := uuid.Parse(pReq.SeatID)
		if err != nil {
			return nil, errors.New("invalid seat_id format: " + pReq.SeatID)
		}

		seat, err := s.repo.GetSeatByID(pReq.SeatID)
		if err != nil {
			return nil, errors.New("seat not found: " + pReq.SeatID)
		}
		if !seat.IsAvailable {
			return nil, errors.New("seat is not available: " + seat.SeatNumber)
		}
		if seat.SeatType != fareType.SeatType {
			return nil, errors.New("seat " + seat.SeatNumber + " type does not match the selected fare class")
		}
		if seat.BusInstanceID != busInstanceID {
			return nil, errors.New("seat " + seat.SeatNumber + " does not belong to the selected bus")
		}

		// Child discount — 50% of fare (configurable per operator; 50% default per spec)
		seatPrice := fareType.Price + seat.ExtraCharge
		if pReq.PassengerType == "child" {
			seatPrice = seatPrice * 0.5
		}
		baseFareTotal += seatPrice

		dob, parseErr := time.Parse("2006-01-02", pReq.DateOfBirth)
		if parseErr != nil {
			return nil, errors.New("invalid date_of_birth format for passenger " + pReq.FirstName + " — expected YYYY-MM-DD")
		}

		// First passenger is primary by default
		isPrimary := i == 0 && !isPrimarySet
		if isPrimary {
			isPrimarySet = true
		}

		passengers = append(passengers, model.Passenger{
			SeatID:        &seatUUID,
			FirstName:     pReq.FirstName,
			LastName:      pReq.LastName,
			DateOfBirth:   dob,
			Gender:        pReq.Gender,
			PassengerType: pReq.PassengerType,
			IDType:        pReq.IDType,
			IDNumber:      pReq.IDNumber,
			IsPrimary:     isPrimary,
		})
		seatIDs = append(seatIDs, pReq.SeatID)
	}

	// ── Lock seats in Redis (all-or-nothing) ──────────────────────────────────
	cfg := config.LoadConfig()
	expMin, _ := strconv.Atoi(cfg.BOOKING_EXPIRY_MINUTES)
	if expMin <= 0 {
		expMin = 15
	}
	lockTTL := time.Duration(expMin) * time.Minute

	if err := busredis.LockSeats(ctx, s.rdb, req.BusInstanceID, seatIDs, userID, lockTTL); err != nil {
		return nil, errors.New("seat(s) temporarily held by another session — " + err.Error())
	}

	// ── Compute pricing ───────────────────────────────────────────────────────
	taxes := baseFareTotal * 0.05 // 5% GST
	totalAmount := baseFareTotal + taxes

	gstin := ""
	if req.GSTIN != nil {
		gstin = *req.GSTIN
	}

	expiresAt := time.Now().Add(lockTTL)

	booking := &model.Booking{
		PNR:             generatePNR(),
		UserID:          userUUID,
		BusInstanceID:   busInstanceID,
		FareTypeID:      fareTypeID,
		BoardingPointID: boardingID,
		DroppingPointID: droppingID,
		Source:          "allocated",
		SeatType:        fareType.SeatType,
		Status:          "PENDING_PAYMENT",
		BaseFare:        baseFareTotal,
		Taxes:           taxes,
		ServiceFee:      0,
		TotalAmount:     totalAmount,
		Currency:        "INR",
		GSTIN:           gstin,
		ExpiresAt:       &expiresAt,
	}

	if err := s.repo.CreateBooking(booking, passengers); err != nil {
		// Release locks if DB write fails
		_ = busredis.UnlockSeatsByOwner(ctx, s.rdb, req.BusInstanceID, seatIDs, userID)
		return nil, errors.New("failed to create booking: " + err.Error())
	}

	// ── Publish bus.booking.created ───────────────────────────────────────────
	// The future Payment Service listens here and will reply with payment_url via redpanda.
	s.producer.PublishBookingCreated(ctx, redpanda.BookingCreatedEvent{
		BookingID:      booking.ID.String(),
		PNR:            booking.PNR,
		UserID:         userID,
		TotalAmount:    booking.TotalAmount,
		Currency:       booking.Currency,
		PassengerCount: len(passengers),
	})

	return &dto.BookingResponse{
		ID:            booking.ID.String(),
		PNR:           booking.PNR,
		Status:        booking.Status,
		BusInstanceID: booking.BusInstanceID.String(),
		SeatType:      booking.SeatType,
		BaseFare:      booking.BaseFare,
		Taxes:         booking.Taxes,
		ServiceFee:    booking.ServiceFee,
		TotalAmount:   booking.TotalAmount,
		Currency:      booking.Currency,
		BookedAt:      booking.BookedAt,
		ExpiresAt:     booking.ExpiresAt,
		// PaymentURL: Payment Service will be integrated as a separate service.
		// It will listen to bus.booking.created and generate the Stripe checkout URL.
		// For now this is empty — same pattern as the Train Service.
		PaymentURL: "",
	}, nil
}

// ── GetBookingByID ────────────────────────────────────────────────────────────

func (s *bookingService) GetBookingByID(id string, userID string) (*dto.BookingResponse, error) {
	booking, err := s.repo.FindBookingByID(id, userID)
	if err != nil {
		return nil, err
	}
	return bookingToDTO(booking), nil
}

// ── GetBookingByPNR ───────────────────────────────────────────────────────────

func (s *bookingService) GetBookingByPNR(pnr string, userID string) (*dto.BookingResponse, error) {
	booking, err := s.repo.FindBookingByPNR(pnr, userID)
	if err != nil {
		return nil, err
	}
	return bookingToDTO(booking), nil
}

// ── GetUserBookings ───────────────────────────────────────────────────────────

func (s *bookingService) GetUserBookings(userID string) ([]dto.BookingResponse, error) {
	bookings, err := s.repo.FindBookingsByUserID(userID)
	if err != nil {
		return nil, err
	}
	resp := make([]dto.BookingResponse, 0, len(bookings))
	for i := range bookings {
		resp = append(resp, *bookingToDTO(&bookings[i]))
	}
	return resp, nil
}

// ── ConfirmBooking ────────────────────────────────────────────────────────────
//
// Confirms a PENDING_PAYMENT booking. In production this is triggered
// automatically by the redpanda consumer when payment.completed arrives from the
// Payment Service.  The HTTP endpoint remains available for manual/admin use.

func (s *bookingService) ConfirmBooking(id string, userID string) error {
	ctx := context.Background()

	booking, err := s.repo.FindBookingByID(id, userID)
	if err != nil {
		return errors.New("booking not found or access denied")
	}
	if booking.Status != "PENDING_PAYMENT" {
		return errors.New("only PENDING_PAYMENT bookings can be confirmed")
	}
	if booking.ExpiresAt != nil && time.Now().After(*booking.ExpiresAt) {
		_ = s.repo.UpdateBookingStatus(id, userID, "EXPIRED", "")
		return errors.New("booking has expired due to timeout")
	}

	// 1. Update status → CONFIRMED
	if err := s.repo.UpdateBookingStatus(id, userID, "CONFIRMED", ""); err != nil {
		return err
	}

	// 2. Mark seats as unavailable in DB
	seatIDs := extractSeatIDs(booking.Passengers)
	if len(seatIDs) > 0 {
		if err := s.repo.UpdateMultipleSeatsAvailability(seatIDs, false); err != nil {
			return errors.New("failed to lock seats in database: " + err.Error())
		}
		// 3. Release Redis seat locks — seats are now DB-locked
		_ = busredis.UnlockSeatsByOwner(ctx, s.rdb, booking.BusInstanceID.String(), seatIDs, userID)
	}

	if err := s.repo.DecrementInventoryOnConfirm(
		booking.BusInstanceID.String(),
		booking.FareTypeID.String(),
		booking.SeatType,
		len(seatIDs),
	); err != nil {
		log.Printf("[booking-service] Inventory update failed (non-fatal): %v", err)
	}

	// 5. Generate e-ticket (stub — QR Service will be called via gRPC in production)
	ticketNumber := "BUS-" + booking.PNR
	eTicket := &model.ETicket{
		BookingID:    booking.ID,
		TicketNumber: ticketNumber,
		QRCodeURL:    "https://storage.tripneo.com/qr/bus/" + booking.PNR + ".png",
		QRData:       "SIGNED:" + booking.PNR + ":" + booking.ID.String(),
	}
	_ = s.repo.SaveETicket(eTicket)

	// 6. Publish bus.booking.confirmed
	s.producer.PublishBookingConfirmed(ctx, redpanda.BookingConfirmedEvent{
		BookingID:      booking.ID.String(),
		PNR:            booking.PNR,
		UserID:         userID,
		TotalAmount:    booking.TotalAmount,
		TicketNumber:   ticketNumber,
		PassengerCount: len(booking.Passengers),
	})

	// ── Verification: Simulate external payment completion ───────────────────
	s.producer.PublishPaymentCompleted(ctx, redpanda.PaymentCompletedEvent{
		BookingID:  booking.ID.String(),
		PaymentRef: "VERIFIED-" + booking.PNR,
		Amount:     booking.TotalAmount,
	})

	// 7. Notify — Notification Service will send email + WhatsApp with QR code
	s.producer.PublishNotification(ctx, redpanda.NotificationEvent{
		Type:            "booking_confirmed",
		RecipientUserID: userID,
		Template:        "bus_booking_confirmed",
		Data: map[string]interface{}{
			"pnr":           booking.PNR,
			"ticket_number": ticketNumber,
			"total_amount":  booking.TotalAmount,
		},
	})

	return nil
}

// ── CancelBooking ─────────────────────────────────────────────────────────────

func (s *bookingService) CancelBooking(id string, userID string, req *dto.CancelBookingRequest) (*dto.CancelBookingResponse, error) {
	ctx := context.Background()

	booking, err := s.repo.FindBookingByID(id, userID)
	if err != nil {
		return nil, errors.New("booking not found or access denied")
	}
	if booking.Status != "CONFIRMED" && booking.Status != "PENDING_PAYMENT" {
		return nil, errors.New("only CONFIRMED or PENDING_PAYMENT bookings can be cancelled")
	}

	// ── Determine refund based on cancellation policy table ───────────────────
	hoursLeft := int(time.Until(booking.BusInstance.DepartureAt).Hours())
	policy, err := s.repo.GetActiveCancellationPolicy(hoursLeft)
	if err != nil {
		return nil, errors.New("failed to determine refund policy: " + err.Error())
	}

	// If the fare type is non-refundable, override to 0% refund
	refundPct := policy.RefundPercentage
	if !booking.FareType.IsRefundable && booking.Status == "CONFIRMED" {
		refundPct = 0
	}
	refundAmount := booking.TotalAmount * (refundPct / 100)

	// ── Release seats ─────────────────────────────────────────────────────────
	seatIDs := extractSeatIDs(booking.Passengers)
	if len(seatIDs) > 0 {
		if err := s.repo.UpdateMultipleSeatsAvailability(seatIDs, true); err != nil {
			return nil, errors.New("failed to release seats: " + err.Error())
		}
		// Unlock any lingering Redis locks (PENDING_PAYMENT case)
		_ = busredis.UnlockSeatsByOwner(ctx, s.rdb, booking.BusInstanceID.String(), seatIDs, userID)
	}

	// ── Restore inventory ─────────────────────────────────────────────────────
	if booking.Status == "CONFIRMED" {
		_ = s.repo.IncrementInventoryOnCancel(
			booking.BusInstanceID.String(),
			booking.FareTypeID.String(),
			booking.SeatType,
			len(seatIDs),
		)
	}

	// ── Reason ───────────────────────────────────────────────────────────────
	reason := "User requested cancellation"
	if req != nil && req.Reason != "" {
		reason = req.Reason
	}

	// ── Write cancellation record ─────────────────────────────────────────────
	var policyID *uuid.UUID
	if policy.ID != (uuid.UUID{}) {
		pid := policy.ID
		policyID = &pid
	}
	cancelRecord := &model.Cancellation{
		BookingID:       booking.ID,
		Reason:          reason,
		RefundAmount:    refundAmount,
		RefundStatus:    "PENDING",
		PolicyAppliedID: policyID,
	}
	if err := s.repo.CreateCancellation(cancelRecord); err != nil {
		return nil, errors.New("failed to log cancellation: " + err.Error())
	}

	// ── Update booking status → CANCELLED ─────────────────────────────────────
	if err := s.repo.UpdateBookingStatus(id, userID, "CANCELLED", ""); err != nil {
		return nil, err
	}

	// ── Publish bus.booking.cancelled ─────────────────────────────────────────
	// Payment Service listens and initiates Stripe refund for CONFIRMED bookings.
	if booking.Status == "CONFIRMED" {
		s.producer.PublishBookingCancelled(ctx, redpanda.BookingCancelledEvent{
			BookingID:    booking.ID.String(),
			PNR:          booking.PNR,
			UserID:       userID,
			RefundAmount: refundAmount,
			Reason:       reason,
		})

		// ── Verification: Simulate external payment refund ───────────────────
		s.producer.PublishPaymentRefunded(ctx, redpanda.PaymentRefundedEvent{
			BookingID: booking.ID.String(),
			Amount:    refundAmount,
		})
	} else if booking.Status == "PENDING_PAYMENT" {
		// ── Verification: Simulate external payment failure ────────────────────
		s.producer.PublishPaymentFailed(ctx, redpanda.PaymentFailedEvent{
			BookingID: booking.ID.String(),
			Reason:    "MANUAL_CANCELLATION",
		})
	}

	// ── Notify ────────────────────────────────────────────────────────────────
	s.producer.PublishNotification(ctx, redpanda.NotificationEvent{
		Type:            "booking_cancelled",
		RecipientUserID: userID,
		Template:        "bus_booking_cancelled",
		Data: map[string]interface{}{
			"pnr":           booking.PNR,
			"refund_amount": refundAmount,
			"refund_status": "PENDING",
		},
	})

	return &dto.CancelBookingResponse{
		BookingID:    booking.ID.String(),
		PNR:          booking.PNR,
		Status:       "CANCELLED",
		RefundAmount: refundAmount,
		RefundStatus: "PENDING",
	}, nil
}

// ── GetBookingTicket ──────────────────────────────────────────────────────────

func (s *bookingService) GetBookingTicket(id string, userID string) (*dto.TicketResponse, error) {
	ticket, err := s.repo.GetETicketByBookingID(id, userID)
	if err != nil {
		return nil, errors.New("ticket not found or access denied")
	}
	return &dto.TicketResponse{
		BookingID:    ticket.BookingID.String(),
		PNR:          ticket.Booking.PNR,
		TicketNumber: ticket.TicketNumber,
		QRCodeURL:    ticket.QRCodeURL,
		IssuedAt:     ticket.IssuedAt,
	}, nil
}
