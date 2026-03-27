package models

import (
	"time"

	"github.com/google/uuid"
)

// TrainSchedule represents one specific dated run of a train template.
type TrainSchedule struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TrainID      uuid.UUID `gorm:"type:uuid;not null;index"`
	Train        Train     `gorm:"foreignKey:TrainID"`
	ScheduleDate time.Time `gorm:"type:date;not null"`                   // The date the train starts its journey
	DepartureAt  time.Time `gorm:"not null"`                             // Full datetime of departure from the first station
	ArrivalAt    time.Time `gorm:"not null"`                             // Full datetime of arrival at the final station
	Status       string    `gorm:"size:20;not null;default:'SCHEDULED'"` // SCHEDULED|DEPARTED|IN_TRANSIT|ARRIVED|CANCELLED|DELAYED
	DelayMinutes int       `gorm:"default:0"`

	// Global seat availability for the entire run
	AvailableSL  int `gorm:"not null;default:0"`
	Available3AC int `gorm:"not null;default:0"`
	Available2AC int `gorm:"not null;default:0"`
	Available1AC int `gorm:"not null;default:0"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (TrainSchedule) TableName() string {
	return "train_schedules"
}
