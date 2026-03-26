package model

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Bus struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string         `gorm:"type:varchar(150);not null" json:"name"`
	Number      string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"number"`
	Type        string         `gorm:"type:varchar(50);not null" json:"type"`
	Operator    string         `gorm:"type:varchar(150);not null" json:"operator"`
	TotalSeats  int            `gorm:"not null" json:"total_seats"`
	Amenities   string         `gorm:"type:text" json:"amenities"`
	Status      string         `gorm:"type:varchar(30);default:'active'" json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *Bus) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}