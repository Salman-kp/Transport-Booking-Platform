package dto

type PassengerDto struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	DateOfBirth    string `json:"date_of_birth"` // YYYY-MM-DD
	Gender         string `json:"gender"`
	PassengerType  string `json:"passenger_type"` // adult | child | infant
	IDType         string `json:"id_type"`        // PASSPORT | AADHAAR | PAN
	IDNumber       string `json:"id_number"`
	SeatID         string `json:"seat_id,omitempty"`
	MealPreference string `json:"meal_preference,omitempty"`
}

type AncillaryBookDto struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	PassengerID string  `json:"passenger_id,omitempty"`
}

type CreateBookingRequest struct {
	FlightInstanceID string             `json:"flight_instance_id"`
	FareTypeID       string             `json:"fare_type_id"`
	TripType         string             `json:"trip_type"`
	SeatClass        string             `json:"seat_class"`
	Passengers       []PassengerDto     `json:"passengers"`
	Ancillaries      []AncillaryBookDto `json:"ancillaries"`
	GSTIN            string             `json:"gstin,omitempty"`
}

type BookingResponse struct {
	ID                 string             `json:"id"`
	PNR                string             `json:"pnr"`
	FlightInstanceID   string             `json:"flight_instance_id"`
	Source             string             `json:"source"`
	Status             string             `json:"status"`
	SeatClass          string             `json:"seat_class"`
	TripType           string             `json:"trip_type"`
	Passengers         []PassengerDto     `json:"passengers,omitempty"`
	Ancillaries        []AncillaryBookDto `json:"ancillaries,omitempty"`
	BaseFare           float64            `json:"base_fare"`
	Taxes              float64            `json:"taxes"`
	ServiceFee         float64            `json:"service_fee"`
	AncillariesTotal   float64            `json:"ancillaries_total"`
	TotalAmount        float64            `json:"total_amount"`
	Currency           string             `json:"currency"`
	StripeClientSecret string             `json:"stripe_client_secret,omitempty"`
	BookedAt           string             `json:"booked_at"`
	ExpiresAt          string             `json:"expires_at,omitempty"`
	FlightNumber       string             `json:"flight_number,omitempty"`
	Origin             string             `json:"origin,omitempty"`
	Destination        string             `json:"destination,omitempty"`
	DepartureTime      string             `json:"departure_time,omitempty"`
	ArrivalTime        string             `json:"arrival_time,omitempty"`
	RefundAmount       *float64           `json:"refund_amount,omitempty"`
	RefundStatus       string             `json:"refund_status,omitempty"`
}

type CancelBookingRequest struct {
	Reason string `json:"reason,omitempty"`
}

type CancelBookingResponse struct {
	BookingID    string  `json:"booking_id"`
	PNR          string  `json:"pnr"`
	Status       string  `json:"status"`
	RefundAmount float64 `json:"refund_amount"`
	RefundStatus string  `json:"refund_status"`
}

type TicketPassenger struct {
	PassengerName string `json:"passenger_name"`
	SeatNumber    string `json:"seat_number"`
}

type TicketResponse struct {
	BookingID    string                  `json:"booking_id"`
	TicketNumber string                  `json:"ticket_number"`
	QRCodeURL    string                  `json:"qr_code_url"`
	PNR          string                  `json:"pnr"`
	Status       string                  `json:"status"`
	TotalAmount  float64                 `json:"total_amount"`
	RefundAmount *float64                `json:"refund_amount,omitempty"`
	RefundStatus string                  `json:"refund_status,omitempty"`
	Flight       InstanceDetailsResponse `json:"flight"`
	SeatClass    string                  `json:"seat_class"`
	Passengers   []TicketPassenger       `json:"passengers"`
}
