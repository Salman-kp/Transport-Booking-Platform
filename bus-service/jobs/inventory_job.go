package jobs

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/Salman-kp/tripneo/bus-service/seed"
	"gorm.io/gorm"
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
			log.Printf("[CRON ERROR] Invalid DaysOfWeek for bus %s: %v\n", templateBus.BusNumber, err)
			continue
		}

		for i := 0; i < lookaheadDays; i++ {
			targetDate := today.AddDate(0, 0, i)

			targetWeekday := int(targetDate.Weekday())
			if targetWeekday == 0 {
				targetWeekday = 7 // Map Sunday to 7
			}

			if !contains(daysOfWeek, targetWeekday) {
				continue
			}

			if generateForDate(db, templateBus, targetDate) {
				insertedCount++
			}
		}
	}
	log.Printf("[CRON] Expansion completed. %d instances ensured for the next 30 days.\n", insertedCount)
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
	// 1. Idempotency Check (Pre-check instead of OnConflict)
	var existing model.BusInstance
	if err := db.Where("bus_id = ? AND travel_date = ?", bus.ID, targetDate).First(&existing).Error; err == nil {
		return false // Already exists
	}

	// 2. Transactional Operation
	err := db.Transaction(func(tx *gorm.DB) error {
		departureAt, err := combineDateAndTime(bus.BusNumber, targetDate, bus.DepartureTime)
		if err != nil {
			return err
		}
		arrivalAt, err := combineDateAndTime(bus.BusNumber, targetDate, bus.ArrivalTime)
		if err != nil {
			return err
		}

		// 3. Overnight Trip Handling
		// If arrival time is earlier than or equal to departure time, it's an overnight trip
		if arrivalAt.Before(departureAt) || arrivalAt.Equal(departureAt) {
			arrivalAt = arrivalAt.Add(24 * time.Hour)
		}

		// 4. Validation
		if departureAt.Equal(arrivalAt) {
			log.Printf("[CRON ERROR] Invalid schedule for bus %s: Departure and Arrival are identical (%s)\n", bus.BusNumber, departureAt)
			return gorm.ErrInvalidData
		}

		duration := arrivalAt.Sub(departureAt)
		if duration <= 0 {
			log.Printf("[CRON ERROR] Invalid duration (%v) for bus %s on %s\n", duration, bus.BusNumber, targetDate.Format("2006-01-02"))
			return gorm.ErrInvalidData
		}

		// 5. Debug Logging
		log.Printf("[CRON DEBUG] Bus %s on %s: Parsed Dep %s, Arr %s | Final Dep_At: %s, Arr_At: %s (Duration: %v)\n",
			bus.BusNumber, targetDate.Format("2006-01-02"),
			bus.DepartureTime, bus.ArrivalTime,
			departureAt.Format("2006-01-02 15:04:05"), arrivalAt.Format("2006-01-02 15:04:05"),
			duration,
		)

		// 6. Derive Pricing Strategy
		var lastInst model.BusInstance
		basePriceSeater, basePriceSemiSleeper, basePriceSleeper := 0.0, 0.0, 0.0

		// Identify active seat types from layout
		var layout map[string]interface{}
		if err := json.Unmarshal(bus.BusType.SeatLayout, &layout); err == nil {
			if _, ok := layout["seater"]; ok {
				basePriceSeater = 250.0
			}
			if _, ok := layout["semi_sleeper"]; ok {
				basePriceSemiSleeper = 900.0
			}
			if _, ok := layout["sleeper"]; ok {
				basePriceSleeper = 1200.0
			}
		}

		// Override with last instance prices if available
		if err := tx.Where("bus_id = ?", bus.ID).Order("travel_date DESC").First(&lastInst).Error; err == nil {
			if basePriceSeater > 0 {
				basePriceSeater = lastInst.BasePriceSeater
			}
			if basePriceSemiSleeper > 0 {
				basePriceSemiSleeper = lastInst.BasePriceSemiSleeper
			}
			if basePriceSleeper > 0 {
				basePriceSleeper = lastInst.BasePriceSleeper
			}
		}

		instance := model.BusInstance{
			BusID:                   bus.ID,
			TravelDate:              targetDate,
			DepartureAt:             departureAt,
			ArrivalAt:               arrivalAt,
			Status:                  "SCHEDULED",
			BasePriceSeater:         basePriceSeater,
			BasePriceSemiSleeper:    basePriceSemiSleeper,
			BasePriceSleeper:        basePriceSleeper,
			CurrentPriceSeater:      basePriceSeater,
			CurrentPriceSemiSleeper: basePriceSemiSleeper,
			CurrentPriceSleeper:     basePriceSleeper,
		}

		if err := tx.Create(&instance).Error; err != nil {
			log.Printf("[CRON ERROR] Failed to create instance for bus %s on %s: %v\n", bus.BusNumber, targetDate.Format("2006-01-02"), err)
			return err
		}

		// 7. Seat Generation as Source of Truth
		if len(bus.BusType.SeatLayout) > 0 {
			if err := seed.ComputationallyMapSeats(tx, instance.ID, []byte(bus.BusType.SeatLayout)); err != nil {
				log.Printf("[CRON ERROR] Failed to generate seats for bus %s: %v\n", bus.BusNumber, err)
				return err
			}
		}

		// Reload instance to get updated availability counts from seat generator
		if err := tx.First(&instance, instance.ID).Error; err != nil {
			return err
		}

		// 8. Context-Aware Fare Creation
		var fares []model.FareType
		if instance.AvailableSleeper > 0 {
			fares = append(fares, model.FareType{
				BusInstanceID:   instance.ID,
				SeatType:        "sleeper",
				Name:            "GENERAL",
				Price:           instance.BasePriceSleeper,
				IsRefundable:    false,
				CancellationFee: instance.BasePriceSleeper,
				SeatsAvailable:  instance.AvailableSleeper,
			})
			fares = append(fares, model.FareType{
				BusInstanceID:   instance.ID,
				SeatType:        "sleeper",
				Name:            "FLEXI",
				Price:           instance.BasePriceSleeper + 300,
				IsRefundable:    true,
				CancellationFee: 300,
				SeatsAvailable:  instance.AvailableSleeper,
			})
		}
		if instance.AvailableSemiSleeper > 0 {
			fares = append(fares, model.FareType{
				BusInstanceID:   instance.ID,
				SeatType:        "semi_sleeper",
				Name:            "GENERAL",
				Price:           instance.BasePriceSemiSleeper,
				IsRefundable:    false,
				CancellationFee: instance.BasePriceSemiSleeper,
				SeatsAvailable:  instance.AvailableSemiSleeper,
			})
		}
		if instance.AvailableSeater > 0 {
			fares = append(fares, model.FareType{
				BusInstanceID:   instance.ID,
				SeatType:        "seater",
				Name:            "GENERAL",
				Price:           instance.BasePriceSeater,
				IsRefundable:    false,
				CancellationFee: instance.BasePriceSeater,
				SeatsAvailable:  instance.AvailableSeater,
			})
		}

		if len(fares) > 0 {
			if err := tx.Create(&fares).Error; err != nil {
				log.Printf("[CRON ERROR] Failed to create fares for bus %s: %v\n", bus.BusNumber, err)
				return err
			}
		}

		// 9. Route Point Generation (Cloning from template)
		// Find a template instance (latest one that has boarding points)
		var templateInst model.BusInstance
		if err := tx.Joins("JOIN boarding_points ON boarding_points.bus_instance_id = bus_instances.id").
			Where("bus_id = ?", bus.ID).Order("travel_date DESC").First(&templateInst).Error; err == nil {
			
			// Clone Boarding Points
			var bps []model.BoardingPoint
			if err := tx.Where("bus_instance_id = ?", templateInst.ID).Find(&bps).Error; err == nil {
				for _, bp := range bps {
					offset := bp.PickupTime.Sub(templateInst.DepartureAt)
					newBP := model.BoardingPoint{
						BusInstanceID: instance.ID,
						BusStopID:     bp.BusStopID,
						PickupTime:    instance.DepartureAt.Add(offset),
						Landmark:      bp.Landmark,
						SequenceOrder: bp.SequenceOrder,
					}
					tx.Create(&newBP)
				}
			}

			// Clone Dropping Points
			var dps []model.DroppingPoint
			if err := tx.Where("bus_instance_id = ?", templateInst.ID).Find(&dps).Error; err == nil {
				for _, dp := range dps {
					offset := dp.DropTime.Sub(templateInst.DepartureAt)
					newDP := model.DroppingPoint{
						BusInstanceID: instance.ID,
						BusStopID:     dp.BusStopID,
						DropTime:      instance.DepartureAt.Add(offset),
						Landmark:      dp.Landmark,
						SequenceOrder: dp.SequenceOrder,
					}
					tx.Create(&newDP)
				}
			}
		} else {
			// Fallback: Create default boarding (Origin) and dropping (Destination) points
			// This ensures the bus appears in search results for its primary route
			bp := model.BoardingPoint{
				BusInstanceID: instance.ID,
				BusStopID:     bus.OriginStopID,
				PickupTime:    instance.DepartureAt,
				SequenceOrder: 1,
				Landmark:      "Main Terminal",
			}
			tx.Create(&bp)

			dp := model.DroppingPoint{
				BusInstanceID: instance.ID,
				BusStopID:     bus.DestinationStopID,
				DropTime:      instance.ArrivalAt,
				SequenceOrder: 2,
				Landmark:      "Bus Station",
			}
			tx.Create(&dp)
		}

		log.Printf("[CRON SUCCESS] Generated inventory and route for %s on %s\n", bus.BusNumber, targetDate.Format("2006-01-02"))
		return nil
	})

	return err == nil
}

func combineDateAndTime(busNumber string, d time.Time, timeStr string) (time.Time, error) {
	var t time.Time
	var err error

	// 1. Try ISO format (from SeedBuses: 1970-01-01T15:04:05Z)
	t, err = time.Parse("2006-01-02T15:04:05Z", timeStr)
	if err != nil {
		// 2. Try HH:MM:SS
		t, err = time.Parse("15:04:05", timeStr)
		if err != nil {
			// 3. Try HH:MM
			t, err = time.Parse("15:04", timeStr)
			if err != nil {
				log.Printf("[CRON ERROR] Failed to parse time '%s' for bus %s: %v\n", timeStr, busNumber, err)
				return time.Time{}, err
			}
		}
	}

	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), 0, d.Location()), nil
}


