package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Seat struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	BusID      uuid.UUID `gorm:"type:uuid;not null" json:"bus_id"`
	SeatNumber string    `gorm:"type:varchar(20);not null" json:"seat_number"`
	SeatType   string    `gorm:"type:varchar(50);not null" json:"seat_type"`
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Seat) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
