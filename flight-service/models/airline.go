package models

import (
	"time"

	"github.com/google/uuid"
)

type Airline struct {
	ID        uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()"`
	Name      string    `gorm:"not null;size:100"`
	IataCode  string    `gorm:"not null;size:3;unique"`
	LogoUrl   string
	IsActive  bool `gorm:"default:true"`
	CreatedAt time.Time
}
