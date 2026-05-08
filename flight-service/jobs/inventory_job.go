package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
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

func GenerateUpcomingInventory(db *gorm.DB) {
	log.Println("CRON Starting 30-Day Inventory Generation Expansion")

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
	log.Printf("CRON Expansion completed successfully. %d new daily instances generated.\n", insertedCount)
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

	// Normalize if flight traverses past midnight
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
		BasePriceEconomy:     5000.0,
		CurrentPriceEconomy:  5000.0,
		BasePriceBusiness:    15000.0,
		CurrentPriceBusiness: 15000.0,
	}

	var layout SeatLayout
	if err := json.Unmarshal([]byte(flight.AircraftType.SeatLayout), &layout); err == nil {
		if layout.Economy != nil {
			totalEco := layout.Economy.Rows * len(layout.Economy.Columns)
			instance.PlatformQuotaEconomy = int(float64(totalEco) * 0.3)
		}
		if layout.Business != nil {
			totalBus := layout.Business.Rows * len(layout.Business.Columns)
			instance.PlatformQuotaBusiness = int(float64(totalBus) * 0.3)
		}
	}

	// Use the calculated platform quota as the source of truth for availability
	instance.AvailableEconomy = instance.PlatformQuotaEconomy
	instance.AvailableBusiness = instance.PlatformQuotaBusiness

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "flight_id"}, {Name: "flight_date"}},
		DoUpdates: clause.AssignmentColumns([]string{"platform_quota_economy", "platform_quota_business", "available_economy", "available_business"}),
	}).Create(&instance).Error
	if err != nil {
		fmt.Printf("Error creating/updating instance: %v\n", err)
		return false
	}

	if instance.ID == uuid.Nil {
		db.Where("flight_id = ? AND flight_date = ?", flight.ID, targetDate).First(&instance)
	}

	fares := []models.FareType{
		{FlightInstanceID: instance.ID, SeatClass: "ECONOMY", Name: "Saver", Price: instance.BasePriceEconomy, CabinBaggageKg: 7, CheckinBaggageKg: 0, IsRefundable: false},
		{FlightInstanceID: instance.ID, SeatClass: "ECONOMY", Name: "Flexi", Price: instance.BasePriceEconomy + 1500, CabinBaggageKg: 7, CheckinBaggageKg: 15, IsRefundable: true, CancellationFee: 1000},
		{FlightInstanceID: instance.ID, SeatClass: "BUSINESS", Name: "Super Flexi", Price: instance.BasePriceBusiness, CabinBaggageKg: 14, CheckinBaggageKg: 30, IsRefundable: true},
	}
	db.Clauses(clause.OnConflict{DoNothing: true}).Create(&fares)

	// Refresh available seats for our quota
	db.Where("flight_instance_id = ? AND is_available = ?", instance.ID, true).Delete(&models.Seat{})

	var seats []models.Seat
	currentRow := 1

	// Generate Business seats strictly up to PlatformQuotaBusiness
	if layout.Business != nil && instance.PlatformQuotaBusiness > 0 {
		count := 0
		for r := 0; r < layout.Business.Rows && count < instance.PlatformQuotaBusiness; r++ {
			for _, col := range layout.Business.Columns {
				if col == "" {
					continue
				}
				if count >= instance.PlatformQuotaBusiness {
					break
				}
				seats = append(seats, models.Seat{
					FlightInstanceID: instance.ID,
					SeatNumber:       fmt.Sprintf("%d%s", currentRow, col),
					SeatClass:        "BUSINESS",
					IsAvailable:      true,
				})
				count++
			}
			currentRow++
		}
		// Increment currentRow to skip remaining rows in the layout
		remainingRows := layout.Business.Rows - (currentRow - 1)
		currentRow += remainingRows
	}

	// Generate Economy seats strictly up to PlatformQuotaEconomy
	if layout.Economy != nil && instance.PlatformQuotaEconomy > 0 {
		count := 0
		for r := 0; r < layout.Economy.Rows && count < instance.PlatformQuotaEconomy; r++ {
			for _, col := range layout.Economy.Columns {
				if col == "" {
					continue
				}
				if count >= instance.PlatformQuotaEconomy {
					break
				}
				seats = append(seats, models.Seat{
					FlightInstanceID: instance.ID,
					SeatNumber:       fmt.Sprintf("%d%s", currentRow, col),
					SeatClass:        "ECONOMY",
					IsAvailable:      true,
				})
				count++
			}
			currentRow++
		}
	}

	if len(seats) > 0 {
		db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(seats, 50)
	}

	return true
}

func combineDateAndTime(d time.Time, t time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), 0, d.Location())
}
