package models

import (
	"time"

	"github.com/google/uuid"
)

type Passenger struct {
	ID             uuid.UUID  `gorm:"type:uuid;primarykey;default:gen_random_uuid()"`
	BookingID      uuid.UUID  `gorm:"type:uuid;not null"`
	SeatID         *uuid.UUID `gorm:"type:uuid"`
	Seat           Seat       `gorm:"foreignKey:SeatID"`
	FirstName      string     `gorm:"size:100;not null"`
	LastName       string     `gorm:"size:100;not null"`
	DateOfBirth    time.Time  `gorm:"type:date;not null"`
	Gender         string     `gorm:"size:10;not null"`
	PassengerType  string     `gorm:"size:10;not null"`
	IDType         string     `gorm:"size:20;not null"`
	IDNumber       string     `gorm:"size:50;not null"`
	MealPreference *string    `gorm:"size:20"`
	IsPrimary      bool       `gorm:"default:false"`
	CreatedAt      time.Time
}
