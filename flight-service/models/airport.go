package models

import (
	"time"

	"github.com/google/uuid"
)

type Airport struct {
	ID        uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()"`
	Name      string    `gorm:"size:50;not null"`
	IataCode  string    `gorm:"size:3;not null;unique"`
	City      string    `gorm:"size:50;not null"`
	Country   string    `gorm:"size:50;not null"`
	TimeZone  string    `gorm:"size:50;not null"`
	Latitude  float64
	Longitude float64
	CreatedAt time.Time
}
