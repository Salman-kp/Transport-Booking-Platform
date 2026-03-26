package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/internal/model"
	"gorm.io/gorm"
)

// Upsert tracking (create or update by bus_id)
func UpsertBusTracking(tracking *model.BusTracking) error {
	return db.DB.
		Where("bus_id = ?", tracking.BusID).Assign(tracking).
		FirstOrCreate(tracking).Error
}

// Get tracking by bus ID
func FindTrackingByBusID(busID string) (*model.BusTracking, error) {
	var t model.BusTracking
	err := db.DB.Where("bus_id = ?", busID).First(&t).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &t, err
}

// Log QR scan
func CreateQRScan(scan *model.QRScan) error {
	return db.DB.Create(scan).Error
}
