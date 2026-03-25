package repository

import (
	"errors"

	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/internal/model"
	"gorm.io/gorm"
)

// Search buses based on filters
func SearchBuses(src, dest, date, busType string) ([]model.Schedule, error) {
	var schedules []model.Schedule
	q := db.DB.Preload("Bus").Preload("Route").
		Joins("JOIN buses ON buses.id = schedules.bus_id").
		Joins("JOIN routes ON routes.id = schedules.route_id").
		Where("routes.source = ? AND routes.destination = ?", src, dest).
		Where("DATE(schedules.departure_time) = ?", date).
		Where("schedules.available_seats > 0").
		Where("schedules.status = 'active'")

	if busType != "" {
		q = q.Where("buses.type = ?", busType)
	}

	err := q.Find(&schedules).Error
	return schedules, err
}

// Get schedule by ID
func FindScheduleByID(id string) (*model.Schedule, error) {
	var s model.Schedule
	err := db.DB.Preload("Bus").Preload("Route").
		Where("id = ?", id).
		First(&s).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return &s, err
}

// Decrement seats (atomic)
func DecrementAvailableSeats(scheduleID string, count int) error {
	result := db.DB.Model(&model.Schedule{}).
		Where("id = ? AND available_seats >= ?", scheduleID, count).
		UpdateColumn("available_seats", gorm.Expr("available_seats - ?", count))

	if result.RowsAffected == 0 {
		return errors.New("not enough seats available")
	}

	return result.Error
}

// Increment seats (rollback/cancel)
func IncrementAvailableSeats(scheduleID string, count int) error {
	return db.DB.Model(&model.Schedule{}).
		Where("id = ?", scheduleID).
		UpdateColumn("available_seats", gorm.Expr("available_seats + ?", count)).
		Error
}
