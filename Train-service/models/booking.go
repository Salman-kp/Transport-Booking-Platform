package models

import (
	"time"

	"github.com/google/uuid"
)

// TrainBooking is the core booking record.
type TrainBooking struct {
	ID              uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PNR             string        `gorm:"size:6;uniqueIndex;not null"`
	UserID          string        `gorm:"size:36;not null;index"`
	TrainScheduleID uuid.UUID     `gorm:"type:uuid;not null;index"`
	TrainSchedule   TrainSchedule `gorm:"foreignKey:TrainScheduleID"`

	// Missing fields added below to fix service compilation errors
	FromStationID uuid.UUID `gorm:"type:uuid;not null"`
	FromStation   Station   `gorm:"foreignKey:FromStationID"`
	ToStationID   uuid.UUID `gorm:"type:uuid;not null"`
	ToStation     Station   `gorm:"foreignKey:ToStationID"`
	DepartureTime time.Time `gorm:"not null"`
	ArrivalTime   time.Time `gorm:"not null"`

	SeatClass   string    `gorm:"size:5;not null"`
	Status      string    `gorm:"size:30;not null;default:'PENDING_PAYMENT'"`
	BaseFare    float64   `gorm:"type:decimal(10,2);not null"`
	Taxes       float64   `gorm:"type:decimal(10,2);not null;default:0"`
	ServiceFee  float64   `gorm:"type:decimal(10,2);not null;default:0"`
	TotalAmount float64   `gorm:"type:decimal(10,2);not null"`
	Currency    string    `gorm:"size:3;not null;default:'INR'"`
	PaymentRef  string    `gorm:"size:100"`
	BookedAt    time.Time `gorm:"default:now()"`
	ConfirmedAt *time.Time
	CancelledAt *time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (TrainBooking) TableName() string {
	return "train_bookings"
}

// BookingSeat remains the same
type BookingSeat struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BookingID uuid.UUID      `gorm:"type:uuid;not null;index"`
	Booking   TrainBooking   `gorm:"foreignKey:BookingID"`
	SeatID    uuid.UUID      `gorm:"type:uuid;not null"`
	Seat      TrainInventory `gorm:"foreignKey:SeatID"`
}

func (BookingSeat) TableName() string {
	return "booking_seats"
}
