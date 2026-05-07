package models

import (
	"time"

	"github.com/google/uuid"
)

type Booking struct {
	ID                uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()"`
	PNR               string    `gorm:"size:6;uniqueIndex;not null"`
	UserID            uuid.UUID `gorm:"type:uuid;not null"`
	FlightInstanceID  uuid.UUID `gorm:"type:uuid;not null"`
	FareTypeID        uuid.UUID `gorm:"type:uuid;not null"`
	AgentInventoryID  *uuid.UUID `gorm:"type:uuid"`
	Source            string    `gorm:"size:20;not null"`
	TripType          string    `gorm:"size:20;not null"`
	GroupBookingID    *uuid.UUID `gorm:"type:uuid"`
	SeatClass         string    `gorm:"size:20;not null"`
	Status            string    `gorm:"size:30;not null;default:'PENDING_PAYMENT'"`
	BaseFare          float64   `gorm:"type:decimal(10,2);not null"`
	Taxes             float64   `gorm:"type:decimal(10,2);not null"`
	ServiceFee        float64   `gorm:"type:decimal(10,2);not null;default:0"`
	AncillariesTotal  float64   `gorm:"type:decimal(10,2);not null;default:0"`
	TotalAmount       float64   `gorm:"type:decimal(10,2);not null"`
	Currency          string    `gorm:"size:3;not null;default:'INR'"`
	PaymentRef        *string   `gorm:"size:100"`
	GSTIN             *string   `gorm:"size:15"`
	BookedAt          time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	ConfirmedAt       *time.Time
	CancelledAt       *time.Time
	ExpiresAt         *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time

	FlightInstance FlightInstance  `gorm:"foreignKey:FlightInstanceID"`
	FareType       FareType        `gorm:"foreignKey:FareTypeID"`
	AgentInventory *AgentInventory `gorm:"foreignKey:AgentInventoryID"`
	Ticket         *ETicket        `gorm:"foreignKey:BookingID"`
	Passengers     []Passenger     `gorm:"foreignKey:BookingID"`
	Ancillaries    []Ancillary     `gorm:"foreignKey:BookingID"`
}
