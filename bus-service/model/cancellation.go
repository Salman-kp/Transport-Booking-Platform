package model

import (
	"time"

	"github.com/google/uuid"
)

type CancellationPolicy struct {
	ID                   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name                 string    `gorm:"type:varchar(100);not null" json:"name"`
	HoursBeforeDeparture int       `gorm:"not null" json:"hours_before_departure"`
	RefundPercentage     float64   `gorm:"type:decimal(5,2);not null" json:"refund_percentage"`
	CancellationFee      float64   `gorm:"type:decimal(10,2);not null;default:0" json:"cancellation_fee"`
	IsActive             bool      `gorm:"default:true" json:"is_active"`
	CreatedAt            time.Time `gorm:"default:now()" json:"created_at"`
}

type Cancellation struct {
	ID              uuid.UUID           `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BookingID       uuid.UUID           `gorm:"type:uuid;not null;unique" json:"booking_id"`
	Reason          string              `gorm:"type:text" json:"reason"`
	RefundAmount    float64             `gorm:"type:decimal(10,2);not null" json:"refund_amount"`
	RefundStatus    string              `gorm:"type:varchar(20);not null;default:'PENDING'" json:"refund_status"`
	PolicyAppliedID *uuid.UUID          `gorm:"type:uuid" json:"policy_applied_id"`
	RequestedAt     time.Time           `gorm:"default:now()" json:"requested_at"`
	ProcessedAt     *time.Time          `json:"processed_at"`
	CreatedAt       time.Time           `gorm:"default:now()" json:"created_at"`

	Booking       Booking             `gorm:"foreignKey:BookingID" json:"booking"`
	PolicyApplied *CancellationPolicy `gorm:"foreignKey:PolicyAppliedID" json:"policy_applied"`
}
