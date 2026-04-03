package repository

import (
	"fmt"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/db"
	domainerrors "github.com/nabeel-mp/tripneo/train-service/domain_errors"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/gorm"
)

type SearchResult struct {
	ScheduleID         string    `json:"schedule_id"`
	TrainNumber        string    `json:"train_number"`
	TrainName          string    `json:"train_name"`
	OriginStation      string    `json:"origin_station"`
	DestinationStation string    `json:"destination_station"`
	DepartureAt        time.Time `json:"departure_at"`
	ArrivalAt          time.Time `json:"arrival_at"`
	DurationMinutes    int       `json:"duration_minutes"`
	DelayMinutes       int       `json:"delay_minutes"`
	Status             string    `json:"status"`
	Class              string    `json:"class"`
	AvailableSeats     int       `json:"available_seats"`
	Price              float64   `json:"price"`
}

func SearchTrains(origin, destination, class string, date time.Time) ([]SearchResult, error) {
	var results []SearchResult

	availCol := availabilityColumn(class)
	if availCol == "" {
		return nil, fmt.Errorf("invalid class: %s", class)
	}

	err := db.DB.Raw(`
		SELECT
			ts.id                     AS schedule_id,
			t.train_number,
			t.train_name,
			s1.code                   AS origin_station,
			s2.code                   AS destination_station,
			ts.departure_at,
			ts.arrival_at,
			t.duration_minutes,
			ts.delay_minutes,
			ts.status,
			? AS class,
			`+availCol+` AS available_seats,
			(
				SELECT AVG(price) 
				FROM train_inventory 
				WHERE train_schedule_id = ts.id 
				  AND class = ? 
				  AND status = 'AVAILABLE'
			) AS price
		FROM train_schedules ts
		JOIN trains t ON t.id = ts.train_id
		JOIN train_stops ts1 ON ts1.train_id = t.id
		JOIN stations s1 ON s1.id = ts1.station_id
		JOIN train_stops ts2 ON ts2.train_id = t.id
		JOIN stations s2 ON s2.id = ts2.station_id
		WHERE
			s1.code               = ?
			AND s2.code           = ?
			AND ts1.stop_sequence < ts2.stop_sequence
			AND DATE(ts.departure_at) = ?
			AND ts.status         != 'CANCELLED'
			AND t.is_active       = true
			AND `+availCol+`      > 0
		ORDER BY ts.departure_at ASC
	`,
		class, class,
		origin, destination,
		date.Format("2006-01-02"),
	).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}

	return results, nil
}

// GetScheduleByID fetches a single schedule with its parent Train preloaded.
func GetScheduleByID(scheduleID string) (*models.TrainSchedule, error) {
	var schedule models.TrainSchedule
	err := db.DB.
		Preload("Train").
		Preload("Train.Stops", func(db *gorm.DB) *gorm.DB {
			return db.Order("stop_sequence ASC") // Ensure stops are in order
		}).
		Preload("Train.Stops.Station").
		First(&schedule, "id = ?", scheduleID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrScheduleNotFound
		}
		return nil, fmt.Errorf("db error: %w", err)
	}
	return &schedule, nil
}

// GetTrainByID fetches a Train by its UUID.
func GetTrainByID(trainID string) (*models.Train, error) {
	var train models.Train
	err := db.DB.First(&train, "id = ?", trainID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domainerrors.ErrTrainNotFound
		}
		return nil, fmt.Errorf("db error: %w", err)
	}
	return &train, nil
}

func availabilityColumn(class string) string {
	switch class {
	case "SL":
		return "ts.available_sl"
	case "3AC":
		return "ts.available_3ac"
	case "2AC":
		return "ts.available_2ac"
	case "1AC":
		return "ts.available_1ac"
	default:
		return ""
	}
}

func DecrementAvailability(tx *gorm.DB, scheduleID, class string, n int) error {
	col := availabilityColumnRaw(class)
	if col == "" {
		return fmt.Errorf("invalid class: %s", class)
	}
	return tx.Exec(
		"UPDATE train_schedules SET "+col+" = "+col+" - ? WHERE id = ?",
		n, scheduleID,
	).Error
}

func IncrementAvailability(tx *gorm.DB, scheduleID, class string, n int) error {
	col := availabilityColumnRaw(class)
	if col == "" {
		return fmt.Errorf("invalid class: %s", class)
	}
	return tx.Exec(
		"UPDATE train_schedules SET "+col+" = "+col+" + ? WHERE id = ?",
		n, scheduleID,
	).Error
}

func availabilityColumnRaw(class string) string {
	switch class {
	case "SL":
		return "available_sl"
	case "3AC":
		return "available_3ac"
	case "2AC":
		return "available_2ac"
	case "1AC":
		return "available_1ac"
	default:
		return ""
	}
}
