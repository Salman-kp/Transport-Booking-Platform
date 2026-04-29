package model

import (
	"time"

	"github.com/google/uuid"
)

type FareType struct {
	ID              uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BusInstanceID   uuid.UUID `gorm:"type:uuid;not null" json:"bus_instance_id"`
	SeatType        string    `gorm:"type:varchar(20);not null" json:"seat_type"`
	Name            string    `gorm:"type:varchar(50);not null" json:"name"`
	Price           float64   `gorm:"type:decimal(10,2);not null" json:"price"`
	IsRefundable    bool      `gorm:"not null;default:false" json:"is_refundable"`
	CancellationFee float64   `gorm:"type:decimal(10,2);not null;default:0" json:"cancellation_fee"`
	DateChangeFee   float64   `gorm:"type:decimal(10,2);not null;default:0" json:"date_change_fee"`
	SeatsAvailable  int       `gorm:"not null" json:"seats_available"`
	CreatedAt       time.Time `gorm:"default:now()" json:"created_at"`

	BusInstance *BusInstance `gorm:"foreignKey:BusInstanceID" json:"bus_instance,omitempty"`
}

type Seat struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BusInstanceID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_bus_instance_seat" json:"bus_instance_id"`
	SeatNumber    string    `gorm:"type:varchar(5);not null;uniqueIndex:idx_bus_instance_seat" json:"seat_number"`
	SeatType      string    `gorm:"type:varchar(20);not null" json:"seat_type"`
	BerthType     string    `gorm:"type:varchar(10)" json:"berth_type"`
	Position      string    `gorm:"type:varchar(10);not null" json:"position"`
	ExtraCharge   float64   `gorm:"type:decimal(10,2);default:0" json:"extra_charge"`
	IsAvailable   bool      `gorm:"default:true" json:"is_available"`
	Category      string    `gorm:"type:varchar(20);not null;default:'GENERAL'" json:"category"`

	BusInstance *BusInstance `gorm:"foreignKey:BusInstanceID" json:"bus_instance,omitempty"`
}
