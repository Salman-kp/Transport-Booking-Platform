package jobs

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/Salman-kp/tripneo/bus-service/seed"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GenerateUpcomingInventory securely projects 30 days of repeating routes into raw usable table instances.
func GenerateUpcomingInventory(db *gorm.DB) {
	log.Println("[CRON] Starting 30-Day Bus Inventory Generation Expansion...")

	var buses []model.Bus
	if err := db.Preload("BusType").Where("is_active = ?", true).Find(&buses).Error; err != nil {
		log.Println("[CRON ERROR] Failed retrieving base schedules:", err)
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	lookaheadDays := 30
	insertedCount := 0

	for _, templateBus := range buses {
		var daysOfWeek []int
		if err := json.Unmarshal(templateBus.DaysOfWeek, &daysOfWeek); err != nil {
			log.Println("Invalid DaysOfWeek for bus:", templateBus.ID)
			continue // Invalid or empty JSON array
		}

		for i := 0; i < lookaheadDays; i++ {
			targetDate := today.AddDate(0, 0, i)

			targetWeekday := int(targetDate.Weekday())
			if targetWeekday == 0 {
				targetWeekday = 7
			}

			if !contains(daysOfWeek, targetWeekday) {
				continue
			}

			if generateForDate(db, templateBus, targetDate) {
				insertedCount++
			}
		}
	}
	log.Printf("[CRON] Expansion completed successfully. %d new daily instances generated.\n", insertedCount)
}

func contains(arr []int, val int) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func generateForDate(db *gorm.DB, bus model.Bus, targetDate time.Time) bool {
	tDepart, _ := time.Parse("15:04", bus.DepartureTime)
	tArrival, _ := time.Parse("15:04", bus.ArrivalTime)

	departureAt := combineDateAndTime(targetDate, tDepart)
	arrivalAt := combineDateAndTime(targetDate, tArrival)

	// Normalize if trip traverses past midnight
	if arrivalAt.Before(departureAt) {
		arrivalAt = arrivalAt.Add(24 * time.Hour)
	}

	instance := model.BusInstance{
		BusID:                   bus.ID,
		TravelDate:              targetDate,
		DepartureAt:             departureAt,
		ArrivalAt:               arrivalAt,
		Status:                  "SCHEDULED",
		BasePriceSeater:         500.0,
		BasePriceSemiSleeper:    900.0,
		BasePriceSleeper:        1200.0,
		CurrentPriceSeater:      500.0,
		CurrentPriceSemiSleeper: 900.0,
		CurrentPriceSleeper:     1200.0,
	}

	// Attempt insertion bypassing conflict panics identically to flight-service
	err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&instance).Error
	if err != nil {
		return false
	}

	if instance.ID.String() == "00000000-0000-0000-0000-000000000000" || instance.ID.String() == "" {
		return false // Conflicted / already exists
	}

	// Unroll default Fare Types identically mirroring flight-service classes
	fares := []model.FareType{
		{BusInstanceID: instance.ID, SeatType: "sleeper", Name: "GENERAL", Price: instance.BasePriceSleeper, IsRefundable: false, CancellationFee: instance.BasePriceSleeper},
		{BusInstanceID: instance.ID, SeatType: "sleeper", Name: "FLEXI", Price: instance.BasePriceSleeper + 300, IsRefundable: true, CancellationFee: 300},
		{BusInstanceID: instance.ID, SeatType: "semi_sleeper", Name: "GENERAL", Price: instance.BasePriceSemiSleeper, IsRefundable: false},
	}
	db.Create(&fares)

	// Intercept BusType Layout logic identically rolling out the concrete Seat grid
	if len(bus.BusType.SeatLayout) > 0 {
		_ = seed.ComputationallyMapSeats(db, instance.ID, bus.BusType.SeatLayout)
	}

	return true
}

func combineDateAndTime(d time.Time, t time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), 0, 0, d.Location())
}
