package dto

import "time"

// ── Request DTOs ──────────────────────────────────────────────────────────────

// CreateBookingRequest is the body for POST /api/buses/bookings.
type CreateBookingRequest struct {
	BusInstanceID   string             `json:"bus_instance_id"   validate:"required,uuid"`
	FareTypeID      string             `json:"fare_type_id"      validate:"required,uuid"`
	BoardingPointID string             `json:"boarding_point_id" validate:"required,uuid"`
	DroppingPointID string             `json:"dropping_point_id" validate:"required,uuid"`
	Passengers      []PassengerRequest `json:"passengers"        validate:"required,min=1"`
	GSTIN           *string            `json:"gstin,omitempty"`
}

// PassengerRequest is a single passenger in the booking request.
type PassengerRequest struct {
	FirstName     string `json:"first_name"     validate:"required"`
	LastName      string `json:"last_name"      validate:"required"`
	DateOfBirth   string `json:"date_of_birth"  validate:"required"` // YYYY-MM-DD
	Gender        string `json:"gender"         validate:"required,oneof=male female other"`
	PassengerType string `json:"passenger_type" validate:"required,oneof=adult child"`
	IDType        string `json:"id_type"        validate:"required,oneof=PASSPORT AADHAAR PAN"`
	IDNumber      string `json:"id_number"      validate:"required"`
	SeatID        string `json:"seat_id"        validate:"required,uuid"`
}

// CancelBookingRequest is the optional body for POST /api/buses/bookings/:id/cancel.
type CancelBookingRequest struct {
	Reason string `json:"reason,omitempty"`
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

// BookingResponse is returned after a booking is created or fetched.
type BookingResponse struct {
	ID            string             `json:"id"`
	PNR           string             `json:"pnr"`
	Status        string             `json:"status"`
	BusInstanceID string             `json:"bus_instance_id"`
	SeatType      string             `json:"seat_type,omitempty"`
	BoardingPoint string             `json:"boarding_point,omitempty"`
	DroppingPoint string             `json:"dropping_point,omitempty"`
	BaseFare      float64            `json:"base_fare"`
	Taxes         float64            `json:"taxes"`
	ServiceFee    float64            `json:"service_fee"`
	TotalAmount   float64            `json:"total_amount"`
	Currency      string             `json:"currency"`
	BookedAt      time.Time          `json:"booked_at"`
	ExpiresAt     *time.Time         `json:"expires_at,omitempty"`
	Passengers    []PassengerDetails `json:"passengers,omitempty"`
	PaymentRef    string             `json:"payment_ref,omitempty"`
	// PaymentURL is a placeholder for when the Payment Service is integrated.
	// The future Payment Service will listen to bus.booking.created and populate this.
	PaymentURL string `json:"payment_url"`
}

// PassengerDetails is a summary of a passenger returned in booking responses.
type PassengerDetails struct {
	ID            string `json:"id,omitempty"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	PassengerType string `json:"passenger_type"`
	SeatNumber    string `json:"seat_number,omitempty"`
}

// CancelBookingResponse is returned after a successful cancellation.
type CancelBookingResponse struct {
	BookingID    string  `json:"booking_id"`
	PNR          string  `json:"pnr"`
	Status       string  `json:"status"`
	RefundAmount float64 `json:"refund_amount"`
	RefundStatus string  `json:"refund_status"`
}

// TicketResponse is returned by GET /api/buses/bookings/:id/ticket.
type TicketResponse struct {
	BookingID    string    `json:"booking_id"`
	PNR          string    `json:"pnr"`
	TicketNumber string    `json:"ticket_number"`
	QRCodeURL    string    `json:"qr_code_url"`
	IssuedAt     time.Time `json:"issued_at"`
}
