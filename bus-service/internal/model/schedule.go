package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Schedule struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	BusID          uuid.UUID `gorm:"type:uuid;not null" json:"bus_id"`
	RouteID        uuid.UUID `gorm:"type:uuid;not null" json:"route_id"`
	DepartureTime  time.Time `gorm:"not null" json:"departure_time"`
	ArrivalTime    time.Time `gorm:"not null" json:"arrival_time"`
	Price          int64     `gorm:"not null" json:"price"`
	AvailableSeats int       `gorm:"not null" json:"available_seats"`
	Status         string    `gorm:"type:varchar(30);default:'active'" json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Bus            Bus       `gorm:"foreignKey:BusID" json:"bus,omitempty"`
	Route          Route     `gorm:"foreignKey:RouteID" json:"route,omitempty"`
}

func (s *Schedule) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
