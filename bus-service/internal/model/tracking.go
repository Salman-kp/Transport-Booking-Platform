package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type BusTracking struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	BusID     uuid.UUID `gorm:"type:uuid;not null" json:"bus_id"`
	Latitude  float64   `gorm:"type:decimal(10,6);not null" json:"latitude"`
	Longitude float64   `gorm:"type:decimal(10,6);not null" json:"longitude"`
	Speed     float64   `gorm:"type:float" json:"speed"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Inventory struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ScheduleID     uuid.UUID `gorm:"type:uuid;not null" json:"schedule_id"`
	TotalSeats     int       `gorm:"not null" json:"total_seats"`
	AvailableSeats int       `gorm:"not null" json:"available_seats"`
	PurchaseDate   time.Time `gorm:"not null" json:"purchase_date"`
	ValidDate      time.Time `gorm:"not null" json:"valid_date"`
	CreatedAt      time.Time `json:"created_at"`
}

type QRScan struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	BookingID uuid.UUID `gorm:"type:uuid;not null" json:"booking_id"`
	ScannedBy uuid.UUID `gorm:"type:uuid;not null" json:"scanned_by"`
	ScanTime  time.Time `gorm:"autoCreateTime" json:"scan_time"`
	Status    string    `gorm:"type:varchar(30);not null" json:"status"`
}

func (t *BusTracking) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (i *Inventory) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

func (q *QRScan) BeforeCreate(tx *gorm.DB) error {
	if q.ID == uuid.Nil {
		q.ID = uuid.New()
	}
	return nil
}
