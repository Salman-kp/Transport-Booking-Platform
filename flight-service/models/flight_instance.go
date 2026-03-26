package models

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	SCHEDULED Status = "SCHEDULED"
	BOARDING  Status = "BOARDING"
	DEPARTED  Status = "DEPARTED"
	IN_AIR    Status = "IN_AIR"
	LANDED    Status = "LANDED"
	CANCELLED Status = "CANCELLED"
	DELAYED   Status = "DELAYED"
)

type FlightInstance struct {
	ID       uuid.UUID `gorm:"type:uuid;primarykey;default:gen_random_uuid()"`
	FlightID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_flight_date"`

	Icao string //unique identification for tracking

	FlightDate  time.Time `gorm:"type:date;uniqueIndex:idx_flight_date"`
	DepartureAt time.Time `gorm:"type:timestamptz;not null"`
	ArrivalAt   time.Time `gorm:"type:timestamptz;not null"`

	Status Status `gorm:"size:20;not null;default:'SCHEDULED'"`

	DelayMinutes int

	GateNumber string
	Terminal   string

	AvailableEconomy  int `gorm:"not null"`
	AvailableBusiness int `gorm:"not null"`

	BasePriceEconomy     float64 `gorm:"type:decimal(10,2);not null"`
	BasePriceBusiness    float64 `gorm:"type:decimal(10,2);not null"`
	CurrentPriceEconomy  float64 `gorm:"type:decimal(10,2);not null"`
	CurrentPriceBusiness float64 `gorm:"type:decimal(10,2);not null"`

	CreatedAt time.Time
	UpdatedAt time.Time

	// GORM Relationship Bindings
	Flight Flight `gorm:"foreignKey:FlightID"`
}
