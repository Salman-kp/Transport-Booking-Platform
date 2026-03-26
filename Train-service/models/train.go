package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Station struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string    `gorm:"size:100;not null"`
	Code      string    `gorm:"size:10;not null;uniqueIndex"` // e.g., "NDLS"
	City      string    `gorm:"size:100"`
	CreatedAt time.Time
}

type Train struct {
	ID                 uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TrainNumber        string        `gorm:"size:20;uniqueIndex;not null"`
	TrainName          string        `gorm:"size:100;not null"`
	OriginStation      string        `gorm:"size:10;not null"`
	DestinationStation string        `gorm:"size:10;not null"`
	DepartureTime      string        `gorm:"size:10;not null"`
	ArrivalTime        string        `gorm:"size:10;not null"`
	DurationMinutes    int           `gorm:"not null"`
	DaysOfWeek         pq.Int32Array `gorm:"type:integer[];not null"`
	IsActive           bool          `gorm:"default:true"`
	Stops              []TrainStop   `gorm:"foreignKey:TrainID"` // Relationship for preloading
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type TrainStop struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TrainID       uuid.UUID `gorm:"type:uuid;not null;index"`
	StationID     uuid.UUID `gorm:"type:uuid;not null"`
	Station       Station   `gorm:"foreignKey:StationID"`
	StopSequence  int       `gorm:"not null"`  // 1, 2, 3...
	ArrivalTime   string    `gorm:"size:5"`    // HH:MM
	DepartureTime string    `gorm:"size:5"`    // HH:MM
	DayOffset     int       `gorm:"default:0"` // 0 for same day, 1 for next day
	DistanceKm    int       `gorm:"default:0"`
}
