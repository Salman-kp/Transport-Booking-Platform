package models

import (
	"time"

	"github.com/google/uuid"
)

type FareType struct {
	ID               uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()"`
	FlightInstanceID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_instance_fare"`
	SeatClass        string    `gorm:"size:20;not null;uniqueIndex:idx_instance_fare"`
	Name             string    `gorm:"size:50;not null;uniqueIndex:idx_instance_fare"`
	Price            float64   `gorm:"type:decimal(10,2);not null"`
	CabinBaggageKg   int       `gorm:"not null"`
	CheckinBaggageKg int       `gorm:"default:0;not null"`
	IsRefundable     bool      `gorm:"default:false;not null"`
	CancellationFee  float64   `gorm:"type:decimal(10,2);default:0;not null"`
	DateChangeFee    float64   `gorm:"type:decimal(10,2);default:0;not null"`
	SeatsAvailable   int       `gorm:"not null"`
	CreatedAt        time.Time
}
