package redpanda

import "encoding/json"

// ── Topics produced by bus-service ────────────────────────────────────────────
const (
	TopicBusBookingCreated   = "bus.booking.created"
	TopicBusBookingConfirmed = "bus.booking.confirmed"
	TopicBusBookingCancelled = "bus.booking.cancelled"
	TopicBusBookingExpired   = "bus.booking.expired"
	TopicBusPaymentInitiated = "bus.payment.initiated"
	TopicBusSearched         = "bus.searched"
	TopicNotificationSend    = "notification.send"

	// Topics consumed by bus-service (produced by Payment Service)
	TopicPaymentCompleted = "payment.completed"
	TopicPaymentFailed    = "payment.failed"
	TopicPaymentRefunded  = "payment.refunded"
)

// ── Produced events ───────────────────────────────────────────────────────────

// BookingCreatedEvent — published when a booking enters PENDING_PAYMENT.
// The future Payment Service will listen to this to create a Stripe session.
type BookingCreatedEvent struct {
	BookingID      string  `json:"booking_id"`
	PNR            string  `json:"pnr"`
	UserID         string  `json:"user_id"`
	BusNumber      string  `json:"bus_number"`
	Operator       string  `json:"operator"`
	Origin         string  `json:"origin"`
	Destination    string  `json:"destination"`
	DepartureAt    string  `json:"departure_at"`
	TotalAmount    float64 `json:"total_amount"`
	Currency       string  `json:"currency"`
	PassengerCount int     `json:"passenger_count"`
}

// BookingConfirmedEvent — published after payment succeeds and booking is CONFIRMED.
type BookingConfirmedEvent struct {
	BookingID      string  `json:"booking_id"`
	PNR            string  `json:"pnr"`
	UserID         string  `json:"user_id"`
	BusNumber      string  `json:"bus_number"`
	Operator       string  `json:"operator"`
	Origin         string  `json:"origin"`
	Destination    string  `json:"destination"`
	DepartureAt    string  `json:"departure_at"`
	BoardingPoint  string  `json:"boarding_point"`
	DroppingPoint  string  `json:"dropping_point"`
	TotalAmount    float64 `json:"total_amount"`
	TicketNumber   string  `json:"ticket_number"`
	PassengerCount int     `json:"passenger_count"`
}

// BookingCancelledEvent — published when a user cancels a confirmed booking.
type BookingCancelledEvent struct {
	BookingID    string  `json:"booking_id"`
	PNR          string  `json:"pnr"`
	UserID       string  `json:"user_id"`
	RefundAmount float64 `json:"refund_amount"`
	Reason       string  `json:"reason"`
}

// BookingExpiredEvent — published by the expiry background worker.
type BookingExpiredEvent struct {
	BookingID string `json:"booking_id"`
	PNR       string `json:"pnr"`
	UserID    string `json:"user_id"`
}

// PaymentInitiatedEvent — published when the gRPC call to Payment Service is made.
type PaymentInitiatedEvent struct {
	BookingID string  `json:"booking_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

// ── Consumed events (produced by Payment Service) ─────────────────────────────

// PaymentCompletedEvent — consumed to transition booking → CONFIRMED.
type PaymentCompletedEvent struct {
	BookingID  string  `json:"booking_id"`
	PaymentRef string  `json:"payment_ref"`
	Amount     float64 `json:"amount"`
}

// PaymentFailedEvent — consumed to transition booking → FAILED.
type PaymentFailedEvent struct {
	BookingID string `json:"booking_id"`
	Reason    string `json:"reason"`
}

// PaymentRefundedEvent — consumed to mark cancellation refund_status → COMPLETED.
type PaymentRefundedEvent struct {
	BookingID string  `json:"booking_id"`
	Amount    float64 `json:"amount"`
}

// ── Notification event ────────────────────────────────────────────────────────

// NotificationEvent — published to the Notification Service via Kafka.
type NotificationEvent struct {
	Type            string                 `json:"type"`
	RecipientUserID string                 `json:"recipient_user_id"`
	Template        string                 `json:"template"`
	Data            map[string]interface{} `json:"data"`
}

func toJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
