package services

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/junaid9001/tripneo/flight-service/dto"
	"github.com/junaid9001/tripneo/flight-service/kafka"
	"github.com/junaid9001/tripneo/flight-service/models"
	"github.com/junaid9001/tripneo/flight-service/redis"
	"github.com/junaid9001/tripneo/flight-service/repository"
	"github.com/junaid9001/tripneo/flight-service/rpc"
	"github.com/junaid9001/tripneo/flight-service/ws"
)

type BookingService struct {
	repo            *repository.BookingRepository
	payClient       *rpc.PaymentClient
	wsManager       *ws.Manager
	qrPublicBaseURL string
	qrSigningSecret string
	razorpayChan    chan string // used to pass order IDs back to DTO mapper
}

func NewBookingService(repo *repository.BookingRepository, payClient *rpc.PaymentClient, wsManager *ws.Manager, qrPublicBaseURL string, qrSigningSecret string) *BookingService {
	return &BookingService{
		repo:            repo,
		payClient:       payClient,
		wsManager:       wsManager,
		qrPublicBaseURL: qrPublicBaseURL,
		qrSigningSecret: qrSigningSecret,
	}
}

func generatePNR() string {
	b := make([]byte, 3)
	rand.Read(b)
	return "TR" + hex.EncodeToString(b)[:4]
}

func (s *BookingService) CreateBooking(userID string, req *dto.CreateBookingRequest) (*dto.BookingResponse, error) {
	if req.FlightInstanceID == "" || req.FareTypeID == "" {
		return nil, errors.New("mandatory flight or fare ID missing")
	}

	flightInstance, err := s.repo.GetFlightInstanceByID(req.FlightInstanceID)
	if err != nil {
		return nil, errors.New("invalid flight instance")
	}

	_, err = s.repo.GetFareTypeByID(req.FareTypeID)
	if err != nil {
		return nil, errors.New("invalid fare type")
	}

	if len(req.Passengers) > 9 {
		return nil, errors.New("maximum 9 passengers allowed per booking")
	}

	bdUserId, _ := uuid.Parse(userID)
	fiId, _ := uuid.Parse(req.FlightInstanceID)
	ftId, _ := uuid.Parse(req.FareTypeID)

	var baseFare float64 = 0
	if req.SeatClass == "BUSINESS" {
		baseFare = flightInstance.CurrentPriceBusiness
	} else {
		baseFare = flightInstance.CurrentPriceEconomy
	}

	totalBase := baseFare * float64(len(req.Passengers))

	var ancillaries []models.Ancillary
	var ancTotal float64 = 0
	for _, a := range req.Ancillaries {
		ancTotal += a.Price * float64(a.Quantity)
		ancillaries = append(ancillaries, models.Ancillary{
			Type:        a.Type,
			Description: a.Description,
			Price:       a.Price,
			Quantity:    a.Quantity,
		})
	}

	taxes := totalBase * 0.18 // 18% tax
	serviceFee := 500.0
	totalAmount := totalBase + taxes + serviceFee + ancTotal

	var passengers []models.Passenger
	var lockedSeats []string
	ctx := context.Background()

	// safe cleanup for failed transactions
	lockSucceeded := false
	defer func() {
		if !lockSucceeded {
			for _, ls := range lockedSeats {
				_ = redis.ReleaseSeatLock(ctx, ls)
			}
		}
	}()

	for _, p := range req.Passengers {
		dob, _ := time.Parse("2006-01-02", p.DateOfBirth)
		var sId *uuid.UUID = nil

		// infants don't occupy seats
		if p.PassengerType == "infant" {
			sId = nil
		} else if p.SeatID != "" {
			seat, err := s.repo.GetSeatByID(p.SeatID)
			if err != nil || !seat.IsAvailable {
				return nil, errors.New("seat already permanently booked or currently held")
			}

			// 10 minute hold
			acquired, err := redis.AcquireSeatLock(ctx, p.SeatID, userID, 10*time.Minute)
			if err != nil || !acquired {
				return nil, errors.New("seat is currently held by another user")
			}
			lockedSeats = append(lockedSeats, p.SeatID)

			sidVal, _ := uuid.Parse(p.SeatID)
			sId = &sidVal
		}

		var mp *string = nil
		if p.MealPreference != "" {
			mp = &p.MealPreference
		}

		passengers = append(passengers, models.Passenger{
			FirstName:      p.FirstName,
			LastName:       p.LastName,
			DateOfBirth:    dob,
			Gender:         p.Gender,
			PassengerType:  p.PassengerType,
			IDType:         p.IDType,
			IDNumber:       p.IDNumber,
			SeatID:         sId,
			MealPreference: mp,
		})
	}

	pnr := generatePNR()
	expiresAt := time.Now().Add(10 * time.Minute)
	var gstin *string = nil
	if req.GSTIN != "" {
		gstin = &req.GSTIN
	}

	booking := &models.Booking{
		PNR:              pnr,
		UserID:           bdUserId,
		FlightInstanceID: fiId,
		FareTypeID:       ftId,
		Source:           "live", // default to live for now
		TripType:         req.TripType,
		SeatClass:        req.SeatClass,
		Status:           "PENDING_PAYMENT",
		BaseFare:         totalBase,
		Taxes:            taxes,
		ServiceFee:       serviceFee,
		AncillariesTotal: ancTotal,
		TotalAmount:      totalAmount,
		Currency:         "INR",
		GSTIN:            gstin,
		ExpiresAt:        &expiresAt,
		Passengers:       passengers,
		Ancillaries:      ancillaries,
	}

	if err := s.repo.CreateBooking(booking); err != nil {
		return nil, err
	}

	lockSucceeded = true

	// trigger gRPC call to payment service
	if s.payClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		orderID, err := s.payClient.CreateOrder(ctx, booking.ID.String(), booking.TotalAmount, booking.Currency, userID)
		if err != nil {
			log.Printf("Payment RPC Failed: %v", err)
			// we dont fail the booking, user can retry payment later
		} else {
			resp := mapBookingToDTO(booking)
			resp.StripeClientSecret = orderID
			return resp, nil
		}
	}

	return mapBookingToDTO(booking), nil
}

func mapBookingToDTO(booking *models.Booking) *dto.BookingResponse {
	var passengers []dto.PassengerDto
	for _, p := range booking.Passengers {
		var sId string
		if p.SeatID != nil {
			sId = p.SeatID.String()
		}
		var mp string
		if p.MealPreference != nil {
			mp = *p.MealPreference
		}
		passengers = append(passengers, dto.PassengerDto{
			FirstName:      p.FirstName,
			LastName:       p.LastName,
			DateOfBirth:    p.DateOfBirth.Format("2006-01-02"),
			Gender:         p.Gender,
			PassengerType:  p.PassengerType,
			IDType:         p.IDType,
			IDNumber:       p.IDNumber,
			SeatID:         sId,
			MealPreference: mp,
		})
	}

	var ancillaries []dto.AncillaryBookDto
	for _, a := range booking.Ancillaries {
		ancillaries = append(ancillaries, dto.AncillaryBookDto{
			Type:        a.Type,
			Description: a.Description,
			Price:       a.Price,
			Quantity:    a.Quantity,
		})
	}

	var checkedExpiresAt string
	if booking.ExpiresAt != nil {
		checkedExpiresAt = booking.ExpiresAt.Format(time.RFC3339)
	}

	resp := &dto.BookingResponse{
		ID:               booking.ID.String(),
		PNR:              booking.PNR,
		FlightInstanceID: booking.FlightInstanceID.String(),
		Source:           booking.Source,
		Status:           booking.Status,
		SeatClass:        booking.SeatClass,
		TripType:         booking.TripType,
		BaseFare:         booking.BaseFare,
		Taxes:            booking.Taxes,
		ServiceFee:       booking.ServiceFee,
		AncillariesTotal: booking.AncillariesTotal,
		TotalAmount:      booking.TotalAmount,
		Currency:         booking.Currency,
		BookedAt:         booking.BookedAt.Format(time.RFC3339),
		ExpiresAt:        checkedExpiresAt,
		Passengers:       passengers,
		Ancillaries:      ancillaries,
	}

	// Populate flight details if preloaded
	if booking.FlightInstance.Flight.FlightNumber != "" {
		resp.FlightNumber = booking.FlightInstance.Flight.FlightNumber
		resp.Origin = booking.FlightInstance.Flight.OriginAirport.IataCode
		resp.Destination = booking.FlightInstance.Flight.DestinationAirport.IataCode
		resp.DepartureTime = booking.FlightInstance.DepartureAt.Format(time.RFC3339)
		resp.ArrivalTime = booking.FlightInstance.ArrivalAt.Format(time.RFC3339)
	}

	return resp
}

func (s *BookingService) enrichBookingRefundFields(bookingID string, resp *dto.BookingResponse) {
	if resp == nil {
		return
	}

	cancellation, err := s.repo.GetCancellationByBookingID(bookingID)
	if err != nil || cancellation == nil {
		return
	}

	amount := cancellation.RefundAmount
	resp.RefundAmount = &amount
	resp.RefundStatus = cancellation.RefundStatus
}

func (s *BookingService) GetBookingByID(id string) (*dto.BookingResponse, error) {
	booking, err := s.repo.GetBookingByID(id)
	if err != nil {
		return nil, err
	}

	resp := mapBookingToDTO(booking)
	s.enrichBookingRefundFields(booking.ID.String(), resp)
	return resp, nil
}

func (s *BookingService) GetBookingByPNR(pnr string) (*dto.BookingResponse, error) {
	booking, err := s.repo.GetBookingByPNR(pnr)
	if err != nil {
		return nil, err
	}

	resp := mapBookingToDTO(booking)
	s.enrichBookingRefundFields(booking.ID.String(), resp)
	return resp, nil
}

func (s *BookingService) GetBookingsByUserID(userID string) ([]dto.BookingResponse, error) {
	bookings, err := s.repo.GetBookingsByUserID(userID)
	if err != nil {
		return nil, err
	}

	var responses []dto.BookingResponse
	for _, b := range bookings {
		resp := mapBookingToDTO(&b)
		s.enrichBookingRefundFields(b.ID.String(), resp)
		responses = append(responses, *resp)
	}
	return responses, nil
}

func (s *BookingService) InitiatePayment(id string, userID string) (string, error) {
	booking, err := s.repo.GetBookingByID(id)
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
			log.Printf("Payment RPC Failed: %v", err)
			return "", errors.New("failed to initiate payment with stripe gateway")
		}
		return orderID, nil
	}

	return "", errors.New("payment service is currently unavailable")
}

func (s *BookingService) ConfirmBooking(id string, paymentRef string) error {
	booking, err := s.repo.GetBookingByID(id)
	if err != nil {
		return err
	}

	if booking.Status != "PENDING_PAYMENT" {
		return errors.New("booking is not pending payment")
	}

	booking.Status = "CONFIRMED"
	now := time.Now()
	booking.ConfirmedAt = &now
	if paymentRef != "" {
		booking.PaymentRef = &paymentRef
	}

	qrData, err := s.buildQRData(booking)
	if err != nil {
		return err
	}

	eTicket := &models.ETicket{
		BookingID:    booking.ID,
		TicketNumber: "TKT-" + booking.PNR,
		QRCodeURL:    s.buildQRCodeURL(qrData),
		QRData:       qrData,
	}

	if err := s.repo.ConfirmBookingAndSeats(booking, eTicket); err != nil {
		return err
	}

	ctx := context.Background()
	for _, p := range booking.Passengers {
		if p.SeatID != nil {
			_ = redis.ReleaseSeatLock(ctx, p.SeatID.String())
		}
	}

	log.Println("[KAFKA MOCK] Published event: flight.booking.confirmed for PNR:", booking.PNR)
	return nil
}

func (s *BookingService) buildQRData(booking *models.Booking) (string, error) {
	payload := map[string]interface{}{
		"booking_id":         booking.ID.String(),
		"pnr":                booking.PNR,
		"flight_instance_id": booking.FlightInstanceID.String(),
		"user_id":            booking.UserID.String(),
		"issued_at":          time.Now().UTC().Format(time.RFC3339),
		"domain":             "flight",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	secret := strings.TrimSpace(s.qrSigningSecret)
	if secret == "" {
		secret = "dev-insecure-change-me"
	}

	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write(payloadBytes); err != nil {
		return "", err
	}
	signature := mac.Sum(nil)

	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)
	return fmt.Sprintf("v1.%s.%s", encodedPayload, encodedSignature), nil
}

func (s *BookingService) buildQRCodeURL(data string) string {
	base := strings.TrimSpace(s.qrPublicBaseURL)
	if base == "" {
		base = "http://localhost:8080/api/qr/generate"
	}

	u, err := url.Parse(base)
	if err != nil {
		return base + "?data=" + url.QueryEscape(data)
	}

	q := u.Query()
	q.Set("data", data)
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *BookingService) CancelBooking(id string, userID string, req *dto.CancelBookingRequest) (*dto.CancelBookingResponse, error) {
	booking, err := s.repo.GetBookingByID(id)
	if err != nil {
		return nil, err
	}

	if booking.UserID.String() != userID {
		return nil, errors.New("unauthorized cancellation request")
	}

	if booking.Status != "CONFIRMED" {
		return nil, errors.New("cannot cancel a non-confirmed booking")
	}

	refundAmount := s.calculateRefundAmount(booking)
	reason := "User requested"
	if req != nil && req.Reason != "" {
		reason = req.Reason
	}

	refundStatus := "NOT_REQUIRED"
	canInitiateRefund := refundAmount > 0 && s.payClient != nil && booking.PaymentRef != nil && strings.TrimSpace(*booking.PaymentRef) != ""
	if refundAmount > 0 && !canInitiateRefund {
		refundStatus = "MANUAL_REVIEW"
	}
	if canInitiateRefund {
		refundStatus = "PENDING"
	}

	cancelData := &models.Cancellation{
		BookingID:    booking.ID,
		Reason:       &reason,
		RefundAmount: refundAmount,
		RefundStatus: refundStatus,
	}

	if err := s.repo.CreateCancellation(cancelData, booking); err != nil {
		return nil, err
	}

	if canInitiateRefund {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		_, _, refundErr := s.payClient.CreateRefund(
			ctx,
			booking.ID.String(),
			*booking.PaymentRef,
			refundAmount,
			booking.Currency,
			booking.UserID.String(),
			"requested_by_customer",
		)
		if refundErr != nil {
			log.Printf("[REFUND ERROR] Failed to initiate refund for booking %s: %v", booking.ID.String(), refundErr)
			_ = s.repo.UpdateCancellationRefundStatus(booking.ID.String(), "FAILED")
			refundStatus = "FAILED"
		}
	}

	log.Println("[KAFKA] Published cancellation intent for PNR:", booking.PNR)
	return &dto.CancelBookingResponse{
		BookingID:    booking.ID.String(),
		PNR:          booking.PNR,
		Status:       "CANCELLED",
		RefundAmount: refundAmount,
		RefundStatus: refundStatus,
	}, nil
}

func (s *BookingService) calculateRefundAmount(booking *models.Booking) float64 {
	if booking == nil {
		return 0
	}

	// Non-refundable fares are not eligible for refund.
	if !booking.FareType.IsRefundable {
		return 0
	}

	hoursLeft := time.Until(booking.FlightInstance.DepartureAt).Hours()
	refundPct := 0.0

	if booking.ConfirmedAt != nil && time.Since(*booking.ConfirmedAt) <= 2*time.Hour && hoursLeft > 24 {
		refundPct = 100.0
	} else if hoursLeft >= 72 {
		refundPct = 90.0
	} else if hoursLeft >= 24 {
		refundPct = 60.0
	} else if hoursLeft >= 4 {
		refundPct = 25.0
	}

	refundAmount := booking.TotalAmount * refundPct / 100.0
	if refundPct < 100.0 {
		totalCancellationFee := booking.FareType.CancellationFee * float64(len(booking.Passengers))
		refundAmount -= totalCancellationFee
	}

	if refundAmount < 0 {
		refundAmount = 0
	}

	// normalize to 2 decimal places for response consistency.
	return math.Round(refundAmount*100) / 100
}

func (s *BookingService) GetTicket(bookingID string) (*dto.TicketResponse, error) {
	t, err := s.repo.GetETicketByBookingID(bookingID)
	if err != nil {
		return nil, err
	}

	booking, err := s.repo.GetBookingByID(bookingID)
	if err != nil {
		return nil, err
	}

	var passengers []dto.TicketPassenger
	if len(booking.Passengers) > 0 {
		for _, p := range booking.Passengers {
			passengerName := p.FirstName + " " + p.LastName
			seatNumber := "TBA"
			if p.SeatID != nil {
				if seat, err := s.repo.GetSeatByID(p.SeatID.String()); err == nil {
					seatNumber = seat.SeatNumber
				}
			}
			passengers = append(passengers, dto.TicketPassenger{
				PassengerName: passengerName,
				SeatNumber:    seatNumber,
			})
		}
	} else {
		passengers = append(passengers, dto.TicketPassenger{
			PassengerName: "Traveler",
			SeatNumber:    "TBA",
		})
	}

	resp := &dto.TicketResponse{
		BookingID:    t.BookingID.String(),
		TicketNumber: t.TicketNumber,
		QRCodeURL:    t.QRCodeURL,
		PNR:          booking.PNR,
		Status:       booking.Status,
		TotalAmount:  booking.TotalAmount,
		SeatClass:    booking.SeatClass,
		Passengers:   passengers,
		Flight: dto.InstanceDetailsResponse{
			InstanceID:      booking.FlightInstanceID.String(),
			FlightNumber:    booking.FlightInstance.Flight.FlightNumber,
			Origin:          booking.FlightInstance.Flight.OriginAirport.IataCode,
			Destination:     booking.FlightInstance.Flight.DestinationAirport.IataCode,
			DepartureTime:   booking.FlightInstance.DepartureAt.Format(time.RFC3339),
			ArrivalTime:     booking.FlightInstance.ArrivalAt.Format(time.RFC3339),
			DurationMinutes: booking.FlightInstance.Flight.DurationMinutes,
			Status:          string(booking.FlightInstance.Status),
		},
	}

	cancellation, err := s.repo.GetCancellationByBookingID(booking.ID.String())
	if err == nil && cancellation != nil {
		amount := cancellation.RefundAmount
		resp.RefundAmount = &amount
		resp.RefundStatus = cancellation.RefundStatus
	}

	return resp, nil
}
func (s *BookingService) ProcessPaymentEvent(evt kafka.PaymentCompletedEvent) {
	booking, err := s.repo.GetBookingByID(evt.BookingID)
	if err != nil {
		log.Printf("[KAFKA ERROR] ProcessPaymentEvent: Booking not found %s", evt.BookingID)
		return
	}

	if booking.Status != "PENDING_PAYMENT" {
		log.Printf("[KAFKA INFO] ProcessPaymentEvent: Booking %s already in status %s. Skipping.", evt.BookingID, booking.Status)
		return
	}

	// update to CONFIRMED
	if err := s.ConfirmBooking(evt.BookingID, evt.PaymentID); err != nil {
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

func (s *BookingService) ProcessRefundedEvent(evt kafka.PaymentRefundedEvent) {
	if strings.ToLower(evt.Domain) != "flight" {
		return
	}

	if err := s.repo.UpdateCancellationRefundStatus(evt.BookingID, "COMPLETED"); err != nil {
		log.Printf("[KAFKA ERROR] ProcessRefundedEvent: Failed updating refund status for booking %s: %v", evt.BookingID, err)
		return
	}

	log.Printf("[KAFKA INFO] Refund marked COMPLETED for flight booking %s", evt.BookingID)
}

func (s *BookingService) ProcessRefundFailedEvent(evt kafka.PaymentRefundFailedEvent) {
	if strings.ToLower(evt.Domain) != "flight" {
		return
	}

	if err := s.repo.UpdateCancellationRefundStatus(evt.BookingID, "FAILED"); err != nil {
		log.Printf("[KAFKA ERROR] ProcessRefundFailedEvent: Failed updating refund status for booking %s: %v", evt.BookingID, err)
		return
	}

	log.Printf("[KAFKA INFO] Refund marked FAILED for flight booking %s, reason: %s", evt.BookingID, evt.Reason)
}
