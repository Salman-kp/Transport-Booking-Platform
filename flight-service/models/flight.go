package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Flight struct {
	ID                   uuid.UUID     `gorm:"type:uuid;primarykey;default:gen_random_uuid()"`
	FlightNumber         string        `gorm:"size:10;not null"`
	AirlineID            uuid.UUID     `gorm:"type:uuid;not null"`
	AircraftTypeID       uuid.UUID     `gorm:"type:uuid;not null"`
	OriginAirportID      uuid.UUID     `gorm:"type:uuid;not null"`
	DestinationAirportID uuid.UUID     `gorm:"type:uuid;not null"`
	DepartureTime        time.Time     `gorm:"not null"`
	ArrivalTime          time.Time     `gorm:"not null"`
	DurationMinutes      int           `gorm:"not null"`
	DaysOfWeek           pq.Int64Array `gorm:"type:smallint[]"`
	IsActive             bool          `gorm:"not null;default:true"`
	CreatedAt            time.Time

	// GORM Relationship Bindings
	Airline              Airline      `gorm:"foreignKey:AirlineID"`
	OriginAirport        Airport      `gorm:"foreignKey:OriginAirportID"`
	DestinationAirport   Airport      `gorm:"foreignKey:DestinationAirportID"`
	AircraftType         AircraftType `gorm:"foreignKey:AircraftTypeID"`
}
