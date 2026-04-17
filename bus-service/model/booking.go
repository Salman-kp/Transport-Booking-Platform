package model

import (
	"time"

	"github.com/google/uuid"
)

type Booking struct {
	ID                  uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PNR                 string     `gorm:"type:varchar(6);not null;unique" json:"pnr"`
	UserID              uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	BusInstanceID       uuid.UUID  `gorm:"type:uuid;not null" json:"bus_instance_id"`
	FareTypeID          uuid.UUID  `gorm:"type:uuid;not null" json:"fare_type_id"`
	OperatorInventoryID *uuid.UUID `gorm:"type:uuid" json:"operator_inventory_id"`
	BoardingPointID     uuid.UUID  `gorm:"type:uuid;not null" json:"boarding_point_id"`
	DroppingPointID     uuid.UUID  `gorm:"type:uuid;not null" json:"dropping_point_id"`
	Source              string     `gorm:"type:varchar(20);not null;default:'allocated'" json:"source"`
	SeatType            string     `gorm:"type:varchar(20);not null" json:"seat_type"`
	Status              string     `gorm:"type:varchar(30);not null;default:'PENDING_PAYMENT'" json:"status"`
	BaseFare            float64    `gorm:"type:decimal(10,2);not null" json:"base_fare"`
	Taxes               float64    `gorm:"type:decimal(10,2);not null" json:"taxes"`
	ServiceFee          float64    `gorm:"type:decimal(10,2);not null;default:0" json:"service_fee"`
	TotalAmount         float64    `gorm:"type:decimal(10,2);not null" json:"total_amount"`
	Currency            string     `gorm:"type:varchar(3);not null;default:'INR'" json:"currency"`
	PaymentRef          string     `gorm:"type:varchar(100)" json:"payment_ref"`
	GSTIN               string     `gorm:"type:varchar(15)" json:"gstin"`
	BookedAt            time.Time  `gorm:"default:now()" json:"booked_at"`
	ConfirmedAt         *time.Time `json:"confirmed_at"`
	CancelledAt         *time.Time `json:"cancelled_at"`
	ExpiresAt           *time.Time `json:"expires_at"`
	CreatedAt           time.Time  `gorm:"default:now()" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"default:now()" json:"updated_at"`

	BusInstance       BusInstance        `gorm:"foreignKey:BusInstanceID" json:"bus_instance"`
	FareType          FareType           `gorm:"foreignKey:FareTypeID" json:"fare_type"`
	OperatorInventory *OperatorInventory `gorm:"foreignKey:OperatorInventoryID" json:"operator_inventory"`
	BoardingPoint     BoardingPoint      `gorm:"foreignKey:BoardingPointID" json:"boarding_point"`
	DroppingPoint     DroppingPoint      `gorm:"foreignKey:DroppingPointID" json:"dropping_point"`
	Passengers        []Passenger        `gorm:"foreignKey:BookingID" json:"passengers"`
}

type Passenger struct {
	ID            uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BookingID     uuid.UUID  `gorm:"type:uuid;not null" json:"booking_id"`
	SeatID        *uuid.UUID `gorm:"type:uuid" json:"seat_id"`
	FirstName     string     `gorm:"type:varchar(100);not null" json:"first_name"`
	LastName      string     `gorm:"type:varchar(100);not null" json:"last_name"`
	DateOfBirth   time.Time  `gorm:"type:date;not null" json:"date_of_birth"`
	Gender        string     `gorm:"type:varchar(10);not null" json:"gender"`
	PassengerType string     `gorm:"type:varchar(10);not null" json:"passenger_type"`
	IDType        string     `gorm:"type:varchar(20);not null" json:"id_type"`
	IDNumber      string     `gorm:"type:varchar(50);not null" json:"id_number"`
	IsPrimary     bool       `gorm:"default:false" json:"is_primary"`
	CreatedAt     time.Time  `gorm:"default:now()" json:"created_at"`

	Booking Booking `gorm:"foreignKey:BookingID" json:"booking"`
	Seat    *Seat   `gorm:"foreignKey:SeatID" json:"seat"`
}

type ETicket struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BookingID    uuid.UUID `gorm:"type:uuid;not null;unique" json:"booking_id"`
	TicketNumber string    `gorm:"type:varchar(20);not null;unique" json:"ticket_number"`
	QRCodeURL    string    `gorm:"type:text;not null" json:"qr_code_url"`
	QRData       string    `gorm:"type:text;not null" json:"qr_data"` // HMAC signed payload
	IssuedAt     time.Time `gorm:"default:now()" json:"issued_at"`

	Booking Booking `gorm:"foreignKey:BookingID" json:"booking"`
}
