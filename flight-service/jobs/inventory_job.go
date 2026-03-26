package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/junaid9001/tripneo/flight-service/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SeatLayout struct {
	Economy *struct {
		Rows    int      `json:"rows"`
		Columns []string `json:"columns"`
	} `json:"economy"`
	Business *struct {
		Rows    int      `json:"rows"`
		Columns []string `json:"columns"`
	} `json:"business"`
}

// GenerateUpcomingInventory securely projects 30 days of repeating routes into raw usable table instances.

func GenerateUpcomingInventory(db *gorm.DB) {
	log.Println("[CRON] Starting 30-Day Inventory Generation Expansion...")

	var flights []models.Flight
	if err := db.Preload("AircraftType").Where("is_active = ?", true).Find(&flights).Error; err != nil {
		log.Println("[CRON ERROR] Failed retrieving base schedules:", err)
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	lookaheadDays := 30
	insertedCount := 0

	for _, flight := range flights {

		for i := 0; i < lookaheadDays; i++ {
			targetDate := today.AddDate(0, 0, i)

			targetWeekday := int64(targetDate.Weekday())
			if targetWeekday == 0 {
				targetWeekday = 7
			}

			if !contains(flight.DaysOfWeek, targetWeekday) {
				continue
			}

			if generateForDate(db, flight, targetDate) {
				insertedCount++
			}
		}
	}
	log.Printf("[CRON] Expansion completed successfully. %d new daily instances generated.\n", insertedCount)
}

func contains(arr []int64, val int64) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func generateForDate(db *gorm.DB, flight models.Flight, targetDate time.Time) bool {
	departureAt := combineDateAndTime(targetDate, flight.DepartureTime)
	arrivalAt := combineDateAndTime(targetDate, flight.ArrivalTime)

	// Normalize if flight traverses strictly past midnight
	if arrivalAt.Before(departureAt) {
		arrivalAt = arrivalAt.Add(24 * time.Hour)
	}

	instance := models.FlightInstance{
		FlightID:             flight.ID,
		FlightDate:           targetDate,
		DepartureAt:          departureAt,
		ArrivalAt:            arrivalAt,
		Status:               models.SCHEDULED,
		AvailableEconomy:     0,
		AvailableBusiness:    0,
		BasePriceEconomy:     5000.0, // Statically seeded, real logic would query active pricing engine rules
		CurrentPriceEconomy:  5000.0,
		BasePriceBusiness:    15000.0,
		CurrentPriceBusiness: 15000.0,
	}

	var layout SeatLayout
	if err := json.Unmarshal([]byte(flight.AircraftType.SeatLayout), &layout); err == nil {
		if layout.Economy != nil {
			instance.AvailableEconomy = layout.Economy.Rows * len(layout.Economy.Columns)
		}
		if layout.Business != nil {
			instance.AvailableBusiness = layout.Business.Rows * len(layout.Business.Columns)
		}
	}

	err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&instance).Error
	if err != nil {
		return false // Real DB error occurred
	}

	if instance.ID.String() == "00000000-0000-0000-0000-000000000000" || instance.ID.String() == "" {
		return false
	}

	fares := []models.FareType{
		{FlightInstanceID: instance.ID, SeatClass: "ECONOMY", Name: "Saver", Price: instance.BasePriceEconomy, CabinBaggageKg: 7, CheckinBaggageKg: 0, IsRefundable: false},
		{FlightInstanceID: instance.ID, SeatClass: "ECONOMY", Name: "Flexi", Price: instance.BasePriceEconomy + 1500, CabinBaggageKg: 7, CheckinBaggageKg: 15, IsRefundable: true, CancellationFee: 1000},
		{FlightInstanceID: instance.ID, SeatClass: "BUSINESS", Name: "Super Flexi", Price: instance.BasePriceBusiness, CabinBaggageKg: 14, CheckinBaggageKg: 30, IsRefundable: true},
	}
	db.Create(&fares)

	// == STEP 2: SEAT MATRIX UNROLLING ==
	var seats []models.Seat
	currentRow := 1

	if layout.Business != nil {
		for r := 0; r < layout.Business.Rows; r++ {
			for _, col := range layout.Business.Columns {
				if col == "" {
					continue
				}
				seats = append(seats, models.Seat{FlightInstanceID: instance.ID, SeatNumber: fmt.Sprintf("%d%s", currentRow, col), SeatClass: "BUSINESS", IsAvailable: true})
			}
			currentRow++
		}
	}

	if layout.Economy != nil {
		for r := 0; r < layout.Economy.Rows; r++ {
			for _, col := range layout.Economy.Columns {
				if col == "" {
					continue
				}
				seats = append(seats, models.Seat{FlightInstanceID: instance.ID, SeatNumber: fmt.Sprintf("%d%s", currentRow, col), SeatClass: "ECONOMY", IsAvailable: true})
			}
			currentRow++
		}
	}

	if len(seats) > 0 {
		db.CreateInBatches(seats, 50)
	}

	return true
}

func combineDateAndTime(d time.Time, t time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), 0, d.Location())
}
