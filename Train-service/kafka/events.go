package kafka

import "encoding/json"

// Topics produced by train-service
const (
	TopicBookingCreated   = "train.booking.created"
	TopicBookingConfirmed = "train.booking.confirmed"
	TopicBookingCancelled = "train.booking.cancelled"
	TopicBookingExpired   = "train.booking.expired"
	TopicNotificationSend = "notification.send"

	// Topics consumed by train-service
	TopicPaymentCompleted = "payment.completed"
	TopicPaymentFailed    = "payment.failed"
	TopicPaymentRefunded  = "payment.refunded"
)

// BookingCreatedEvent is published when a booking is created with PENDING_PAYMENT.
type BookingCreatedEvent struct {
	BookingID      string  `json:"booking_id"`
	PNR            string  `json:"pnr"`
	UserID         string  `json:"user_id"`
	TrainName      string  `json:"train_name"`
	TrainNumber    string  `json:"train_number"`
	From           string  `json:"from"`
	To             string  `json:"to"`
	Departure      string  `json:"departure"`
	Class          string  `json:"class"`
	TotalAmount    float64 `json:"total_amount"`
	PassengerCount int     `json:"passenger_count"`
}

// BookingConfirmedEvent is published after payment.completed is consumed.
type BookingConfirmedEvent struct {
	BookingID   string  `json:"booking_id"`
	PNR         string  `json:"pnr"`
	UserID      string  `json:"user_id"`
	TrainName   string  `json:"train_name"`
	TrainNumber string  `json:"train_number"`
	From        string  `json:"from"`
	To          string  `json:"to"`
	Departure   string  `json:"departure"`
	TotalAmount float64 `json:"total_amount"`
}

// BookingCancelledEvent is published when a user cancels.
type BookingCancelledEvent struct {
	BookingID    string  `json:"booking_id"`
	PNR          string  `json:"pnr"`
	UserID       string  `json:"user_id"`
	RefundAmount float64 `json:"refund_amount"`
	Reason       string  `json:"reason"`
}

// BookingExpiredEvent is published by the expiry worker.
type BookingExpiredEvent struct {
	BookingID string `json:"booking_id"`
	PNR       string `json:"pnr"`
	UserID    string `json:"user_id"`
}

// PaymentCompletedEvent is consumed from Payment Service.
type PaymentCompletedEvent struct {
	BookingID  string  `json:"booking_id"`
	PaymentRef string  `json:"payment_ref"`
	Amount     float64 `json:"amount"`
}

// PaymentFailedEvent is consumed from Payment Service.
type PaymentFailedEvent struct {
	BookingID string `json:"booking_id"`
	Reason    string `json:"reason"`
}

// PaymentRefundedEvent is consumed from Payment Service.
type PaymentRefundedEvent struct {
	BookingID string  `json:"booking_id"`
	Amount    float64 `json:"amount"`
}

// NotificationEvent is published to the Notification Service.
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
