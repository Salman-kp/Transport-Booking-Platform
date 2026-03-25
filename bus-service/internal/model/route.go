package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Route struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Source        string    `gorm:"type:varchar(150);not null" json:"source"`
	Destination   string    `gorm:"type:varchar(150);not null" json:"destination"`
	DistanceKm    int       `gorm:"not null" json:"distance_km"`
	EstimatedTime int       `gorm:"not null" json:"estimated_time"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (r *Route) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
