package repository

import (
	"fmt"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/db"
	domainerrors "github.com/nabeel-mp/tripneo/train-service/domain_errors"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/gorm"
)

// SearchResult is what the search handler returns to the client.
// It combines Train + Schedule + available seat count.
type SearchResult struct {
	ScheduleID     string    `json:"schedule_id"`
	TrainNumber    string    `json:"train_number"`
	TrainName      string    `json:"train_name"`
	OriginStation  string    `json:"origin_station"`
	DestStation    string    `json:"destination_station"`
	DepartureAt    time.Time `json:"departure_at"`
	ArrivalAt      time.Time `json:"arrival_at"`
	DurationMins   int       `json:"duration_minutes"`
	DelayMinutes   int       `json:"delay_minutes"`
	Status         string    `json:"status"`
	Class          string    `json:"class"`
	AvailableSeats int       `json:"available_seats"`
	Price          float64   `json:"price"`
}

// SearchTrains queries schedules + inventory for a route and date.
// Only returns schedules that have available_seats > 0 for the requested class.
// No waitlist — if a seat class is full, it is not returned.
func SearchTrains(origin, destination, class string, date time.Time) ([]SearchResult, error) {
	var results []SearchResult

	// Determine which availability column to check based on class
	availCol := availabilityColumn(class)
	if availCol == "" {
		return nil, fmt.Errorf("invalid class: %s", class)
	}

	err := db.DB.Raw(`
		SELECT
			ts.id                     AS schedule_id,
			t.train_number,
			t.train_name,
			t.origin_station,
			t.destination_station,
			ts.departure_at,
			ts.arrival_at,
			t.duration_minutes,
			ts.delay_minutes,
			ts.status,
			? AS class,
			`+availCol+` AS available_seats,
			COALESCE(
				AVG(ti.price) FILTER (WHERE ti.class = ? AND ti.status = 'AVAILABLE'),
				0
			) AS price
		FROM train_schedules ts
		JOIN trains t ON t.id = ts.train_id
		LEFT JOIN train_inventory ti ON ti.train_schedule_id = ts.id
		WHERE
			t.origin_station      = ?
			AND t.destination_station = ?
			AND DATE(ts.departure_at) = ?
			AND ts.status         != 'CANCELLED'
			AND t.is_active       = true
			AND `+availCol+`      > 0
		GROUP BY ts.id, t.id
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
	err := db.DB.Preload("Train").First(&schedule, "id = ?", scheduleID).Error
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

// availabilityColumn maps a class string to the correct
// availability count column on train_schedules.
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

// DecrementAvailability reduces the available seat count for a class
// on a schedule by n. Used after booking confirmation.
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

// IncrementAvailability restores the available seat count for a class.
// Used on cancellation or expiry.
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
