package models

import (
	"github.com/google/uuid"
)

type Seat struct {
	ID               uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()"`
	FlightInstanceID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_instance_seat"`
	SeatNumber       string    `gorm:"size:5;not null;uniqueIndex:idx_instance_seat"`
	SeatClass        string    `gorm:"size:20;not null"`
	Position         string    `gorm:"size:10;not null"`
	ExtraCharge      float64   `gorm:"type:decimal(10,2);default:0"`
	IsAvailable      bool      `gorm:"default:true"`
}
