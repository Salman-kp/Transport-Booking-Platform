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
	"github.com/Salman-kp/tripneo/bus-service/kafka"
	"github.com/Salman-kp/tripneo/bus-service/model"
	busredis "github.com/Salman-kp/tripneo/bus-service/redis"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	"github.com/Salman-kp/tripneo/bus-service/rpc"
	"github.com/Salman-kp/tripneo/bus-service/ws"
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
	InitiatePayment(id string, userID string) (string, error)
	ProcessPaymentEvent(evt kafka.PaymentCompletedEvent)
}

type bookingService struct {
	repo      repository.BookingRepository
	rdb       *goredis.Client
	payClient *rpc.PaymentClient
	wsManager *ws.Manager
}

// NewBookingService constructs a BookingService.
func NewBookingService(repo repository.BookingRepository, rdb *goredis.Client, payClient *rpc.PaymentClient, wsManager *ws.Manager) BookingService {
	return &bookingService{
		repo:      repo,
		rdb:       rdb,
		payClient: payClient,
		wsManager: wsManager,
	}
}

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
		var seatUUIDPtr *uuid.UUID
		if pReq.SeatID != "" {
			seatUUID, err := uuid.Parse(pReq.SeatID)
			if err != nil {
				return nil, errors.New("invalid seat_id format: " + pReq.SeatID)
			}
			seatUUIDPtr = &seatUUID

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

			// Gender validation based on seat category
			genderUpper := strings.ToUpper(pReq.Gender)
			if seat.Category == "WOMEN" && genderUpper != "WOMEN" {
				return nil, errors.New("seat " + seat.SeatNumber + " is reserved for women")
			}
			if seat.Category == "MEN" && genderUpper != "MEN" {
				return nil, errors.New("seat " + seat.SeatNumber + " is reserved for men")
			}

			// Child discount — 50% of fare (configurable per operator; 50% default per spec)
			seatPrice := fareType.Price + seat.ExtraCharge
			if pReq.PassengerType == "child" {
				seatPrice = seatPrice * 0.5
			}
			baseFareTotal += seatPrice
			seatIDs = append(seatIDs, pReq.SeatID)
		} else {
			// Without a seat, we might still charge base fare or a percentage for child/infant
			// Assuming infant on lap pays 10% or something, but let's default to child rules without extra charge
			seatPrice := fareType.Price
			if pReq.PassengerType == "child" || pReq.PassengerType == "infant" {
				seatPrice = seatPrice * 0.5
			}
			baseFareTotal += seatPrice
		}

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
			SeatID:        seatUUIDPtr,
			FirstName:     pReq.FirstName,
			LastName:      pReq.LastName,
			DateOfBirth:   dob,
			Gender:        pReq.Gender,
			PassengerType: pReq.PassengerType,
			IDType:        pReq.IDType,
			IDNumber:      pReq.IDNumber,
			IsPrimary:     isPrimary,
		})
	}

	// ── Deduplicate seatIDs defensively against bad client payloads ───────────
	seen := make(map[string]struct{})
	uniqueSeatIDs := seatIDs[:0]
	for _, id := range seatIDs {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			uniqueSeatIDs = append(uniqueSeatIDs, id)
		}
	}
	seatIDs = uniqueSeatIDs

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

	// ── Trigger gRPC call to payment service ──────────────────────────────────
	var paymentURL string
	if s.payClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		orderID, err := s.payClient.CreateOrder(ctx, booking.ID.String(), booking.TotalAmount, booking.Currency, userID)
		if err != nil {
			log.Printf("Payment gRPC Failed: %v", err)
			// we don't fail the booking, user can retry or wait for auto-expiry
		} else {
			paymentURL = orderID // Stripe client secret
		}
	}

	fullBooking, err := s.GetBookingByID(booking.ID.String(), userID)
	if err == nil {
		fullBooking.PaymentURL = paymentURL
		return fullBooking, nil
	}

	resp := &dto.BookingResponse{
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
		PaymentURL:    paymentURL,
	}

	return resp, nil
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

func (s *bookingService) InitiatePayment(id string, userID string) (string, error) {
	booking, err := s.repo.FindBookingByID(id, userID)
	if err != nil {
		return "", err
	}

	if booking.Status != "PENDING_PAYMENT" {
		return "", errors.New("booking is not pending payment")
	}

	// trigger gRPC call to payment service to get client secret
	if s.payClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		orderID, err := s.payClient.CreateOrder(ctx, booking.ID.String(), booking.TotalAmount, booking.Currency, userID)
		if err != nil {
			log.Printf("Payment gRPC Failed: %v", err)
			return "", errors.New("failed to initiate payment with stripe gateway")
		}
		return orderID, nil
	}

	return "", errors.New("payment service is currently unavailable")
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

	// 5. Generate e-ticket
	ticketNumber := "BUS-" + booking.PNR
	eTicket := &model.ETicket{
		BookingID:    booking.ID,
		TicketNumber: ticketNumber,
		QRCodeURL:    "https://storage.tripneo.com/qr/bus/" + booking.PNR + ".png",
		QRData:       "SIGNED:" + booking.PNR + ":" + booking.ID.String(),
	}
	_ = s.repo.SaveETicket(eTicket)

	log.Println("[KAFKA MOCK] Published event: bus.booking.confirmed for PNR:", booking.PNR)
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
	if booking.Status == "PENDING_PAYMENT" {
		refundAmount = 0
	}

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

	log.Println("[KAFKA MOCK] Published event: bus.booking.cancelled for PNR:", booking.PNR)

	// notify frontend via websocket
	if s.wsManager != nil {
		msg := map[string]interface{}{
			"event": "BOOKING_CANCELLED",
			"payload": map[string]interface{}{
				"booking_id":    booking.ID.String(),
				"pnr":           booking.PNR,
				"refund_amount": refundAmount,
				"status":        "CANCELLED",
			},
		}
		_ = s.wsManager.SendToUser(booking.UserID.String(), msg)
		log.Printf("[WS SUCCESS] Notified user %s of cancelled booking %s", booking.UserID.String(), booking.ID.String())
	}

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

func (s *bookingService) ProcessPaymentEvent(evt kafka.PaymentCompletedEvent) {
	booking, err := s.repo.FindBookingByID(evt.BookingID, evt.UserID)
	if err != nil {
		log.Printf("[KAFKA ERROR] ProcessPaymentEvent: Booking not found %s", evt.BookingID)
		return
	}

	if booking.Status != "PENDING_PAYMENT" {
		log.Printf("[KAFKA INFO] ProcessPaymentEvent: Booking %s already in status %s. Skipping.", evt.BookingID, booking.Status)
		return
	}

	// update to CONFIRMED
	if err := s.ConfirmBooking(evt.BookingID, evt.UserID); err != nil {
		log.Printf("[KAFKA ERROR] ProcessPaymentEvent: Failed to confirm booking %s: %v", evt.BookingID, err)
		return
	}

	// notify frontend via websocket
	if s.wsManager != nil {
		msg := map[string]interface{}{
			"event": "BOOKING_CONFIRMED",
			"payload": map[string]interface{}{
				"booking_id": evt.BookingID,
				"pnr":        booking.PNR,
				"amount":     evt.Amount,
				"currency":   evt.Currency,
				"status":     "CONFIRMED",
			},
		}
		_ = s.wsManager.SendToUser(booking.UserID.String(), msg)
		log.Printf("[WS SUCCESS] Notified user %s of confirmed booking %s", booking.UserID.String(), evt.BookingID)
	}
}
