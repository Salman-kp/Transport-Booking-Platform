package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Booking struct {
	ID            uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        uuid.UUID     `gorm:"type:uuid;not null" json:"user_id"`
	ScheduleID    uuid.UUID     `gorm:"type:uuid;not null" json:"schedule_id"`
	TotalAmount   int64         `gorm:"not null" json:"total_amount"`
	Status        string        `gorm:"type:varchar(30);not null" json:"status"`
	PaymentStatus string        `gorm:"type:varchar(30);not null" json:"payment_status"`
	PaymentRefID  string        `gorm:"type:varchar(150)" json:"payment_ref_id"`
	QRCode        string        `gorm:"type:text;not null" json:"qr_code"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	Schedule      Schedule      `gorm:"foreignKey:ScheduleID" json:"schedule,omitempty"`
	BookingSeats  []BookingSeat `gorm:"foreignKey:BookingID" json:"booking_seats,omitempty"`
}

type BookingSeat struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	BookingID       uuid.UUID `gorm:"type:uuid;not null" json:"booking_id"`
	SeatID          uuid.UUID `gorm:"type:uuid;not null" json:"seat_id"`
	PassengerName   string    `gorm:"type:varchar(150);not null" json:"passenger_name"`
	PassengerAge    int       `gorm:"not null" json:"passenger_age"`
	PassengerGender string    `gorm:"type:varchar(20);not null" json:"passenger_gender"`
	CreatedAt       time.Time `json:"created_at"`
	Seat            Seat      `gorm:"foreignKey:SeatID" json:"seat,omitempty"`
}

func (b *Booking) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

func (bs *BookingSeat) BeforeCreate(tx *gorm.DB) error {
	if bs.ID == uuid.Nil {
		bs.ID = uuid.New()
	}
	return nil
}
