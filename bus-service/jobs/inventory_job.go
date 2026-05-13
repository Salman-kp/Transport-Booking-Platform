package jobs

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GenerateUpcomingInventory projects 30 days of repeating bus templates into BusInstance records.
func GenerateUpcomingInventory(db *gorm.DB) {
	log.Println("[CRON] Starting 30-Day Bus Inventory Generation Expansion...")

	var buses []model.Bus
	if err := db.Preload("BusType").Preload("Operator").Where("is_active = ?", true).Find(&buses).Error; err != nil {
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
				targetWeekday = 7 // Map Sunday → 7
			}

			if !contains(daysOfWeek, targetWeekday) {
				continue
			}

			if generateForDate(db, templateBus, targetDate, targetWeekday) {
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

// ─────────────────────────────────────────────────────────────────────────────
// Prebooking Configuration
// ─────────────────────────────────────────────────────────────────────────────

// prebookingConfig holds per-category seat totals and active (purchased) counts.
type prebookingConfig struct {
	seatType      string
	basePrice     float64
	womenTotal    int
	menTotal      int
	generalTotal  int
	womenActive   int
	menActive     int
	generalActive int
}

func (c prebookingConfig) totalSeats() int  { return c.womenTotal + c.menTotal + c.generalTotal }
func (c prebookingConfig) activeSeats() int { return c.womenActive + c.menActive + c.generalActive }

// resolvePrebookingConfig returns seat counts and base price for the given seat type and day type.
//
// Weekend (days 6,7) → ~60% purchased; Weekday → ~40% purchased.
// All seats are generated; only purchased seats have is_available = true.
//
// Seater (40 total: 8 rows × 5 cols [2+3]):
//   - Categories: first 8 → WOMEN, next 8 → MEN, remaining 24 → GENERAL
//   - 60%: 5W + 5M + 14G = 24 active
//   - 40%: 3W + 3M + 10G = 16 active
//
// Semi-Sleeper (32 total: 8 rows × 4 cols [2+2]):
//   - Categories: first 8 → WOMEN, next 8 → MEN, remaining 16 → GENERAL
//   - 60%: 4W + 4M + 11G = 19 active
//   - 40%: 3W + 3M + 7G  = 13 active
//
// Sleeper (16 berths: 4 rows × 2 sides × 2 berths [lower+upper]), all GENERAL:
//   - 60%: 10 active
//   - 40%: 6 active
func resolvePrebookingConfig(seatType string, isWeekend bool) prebookingConfig {
	cfg := prebookingConfig{seatType: seatType}
	switch seatType {
	case "seater":
		cfg.basePrice = 250.0
		cfg.womenTotal, cfg.menTotal, cfg.generalTotal = 8, 8, 24
		if isWeekend {
			cfg.womenActive, cfg.menActive, cfg.generalActive = 5, 5, 14
		} else {
			cfg.womenActive, cfg.menActive, cfg.generalActive = 3, 3, 10
		}
	case "semi_sleeper":
		cfg.basePrice = 900.0
		cfg.womenTotal, cfg.menTotal, cfg.generalTotal = 8, 8, 16
		if isWeekend {
			cfg.womenActive, cfg.menActive, cfg.generalActive = 4, 4, 11
		} else {
			cfg.womenActive, cfg.menActive, cfg.generalActive = 3, 3, 7
		}
	case "sleeper":
		cfg.basePrice = 1200.0
		cfg.generalTotal = 16 // 4 rows × 2 sides × 2 berths (L+U)
		if isWeekend {
			cfg.generalActive = 10
		} else {
			cfg.generalActive = 6
		}
	}
	return cfg
}

// ─────────────────────────────────────────────────────────────────────────────
// Main generation function
// ─────────────────────────────────────────────────────────────────────────────

func generateForDate(db *gorm.DB, bus model.Bus, targetDate time.Time, weekday int) bool {
	// 1. Idempotency — skip if instance already exists
	var existing model.BusInstance
	if err := db.Where("bus_id = ? AND travel_date = ?", bus.ID, targetDate).First(&existing).Error; err == nil {
		return false
	}

	isWeekend := weekday == 6 || weekday == 7

	// 2. Detect seat type from layout JSON
	var layout map[string]interface{}
	if err := json.Unmarshal(bus.BusType.SeatLayout, &layout); err != nil {
		log.Printf("[CRON ERROR] Failed to parse SeatLayout for bus %s: %v\n", bus.BusNumber, err)
		return false
	}
	seatType := ""
	for k := range layout {
		if k == "seater" || k == "semi_sleeper" || k == "sleeper" {
			seatType = k
			break
		}
	}
	if seatType == "" {
		log.Printf("[CRON ERROR] No recognised seat type in layout for bus %s\n", bus.BusNumber)
		return false
	}

	cfg := resolvePrebookingConfig(seatType, isWeekend)
	sellingPrice := cfg.basePrice + (cfg.basePrice * 30 / 100) // +30% markup

	// 3. Transactional block
	err := db.Transaction(func(tx *gorm.DB) error {
		departureAt, err := combineDateAndTime(bus.BusNumber, targetDate, bus.DepartureTime)
		if err != nil {
			return err
		}
		arrivalAt, err := combineDateAndTime(bus.BusNumber, targetDate, bus.ArrivalTime)
		if err != nil {
			return err
		}

		// Overnight trip: push arrival to next day
		if arrivalAt.Before(departureAt) || arrivalAt.Equal(departureAt) {
			arrivalAt = arrivalAt.Add(24 * time.Hour)
		}

		// Validate duration
		if departureAt.Equal(arrivalAt) || arrivalAt.Sub(departureAt) <= 0 {
			log.Printf("[CRON ERROR] Invalid schedule for bus %s on %s\n", bus.BusNumber, targetDate.Format("2006-01-02"))
			return gorm.ErrInvalidData
		}

		log.Printf("[CRON] Bus %s on %s | type=%s weekend=%v active=%d/%d base=%.0f selling=%.0f\n",
			bus.BusNumber, targetDate.Format("2006-01-02"),
			seatType, isWeekend, cfg.activeSeats(), cfg.totalSeats(), cfg.basePrice, sellingPrice)

		// ── Step A: Create BusInstance ─────────────────────────────────────────
		instance := model.BusInstance{
			BusID:                 bus.ID,
			TravelDate:            targetDate,
			DepartureAt:           departureAt,
			ArrivalAt:             arrivalAt,
			Status:                "SCHEDULED",
			PurchasedPriceOfSeats: cfg.basePrice,
		}
		switch seatType {
		case "seater":
			instance.BasePriceSeater = sellingPrice
			instance.CurrentPriceSeater = sellingPrice
		case "semi_sleeper":
			instance.BasePriceSemiSleeper = sellingPrice
			instance.CurrentPriceSemiSleeper = sellingPrice
		case "sleeper":
			instance.BasePriceSleeper = sellingPrice
			instance.CurrentPriceSleeper = sellingPrice
		}
		if err := tx.Create(&instance).Error; err != nil {
			log.Printf("[CRON ERROR] Failed to create instance for bus %s: %v\n", bus.BusNumber, err)
			return err
		}

		// ── Step B: PrebookingAccounting (before seats) ────────────────────────
		accounting := model.PrebookingAccounting{
			InstanceID:          instance.ID,
			OperatorName:        bus.Operator.Name,
			BusNumber:           bus.BusNumber,
			DepartureDateTime:   instance.DepartureAt,
			ArrivalDateTime:     instance.ArrivalAt,
			TotalPurchasedSeats: cfg.activeSeats(),
			OriginalSeatPrice:   cfg.basePrice,
			SellingSeatPrice:    sellingPrice,
			SpendAmountTotal:    float64(cfg.activeSeats()) * cfg.basePrice,
		}
		if err := tx.Create(&accounting).Error; err != nil {
			log.Printf("[CRON ERROR] Failed to create accounting for bus %s: %v\n", bus.BusNumber, err)
			return err
		}

		// ── Step C: Generate ALL seats (all is_available=true by DB default) ──────
		seats := buildSeats(instance.ID, cfg)
		if len(seats) > 0 {
			if err := tx.Create(&seats).Error; err != nil {
				log.Printf("[CRON ERROR] Failed to insert seats for bus %s: %v\n", bus.BusNumber, err)
				return err
			}
		}

		// ── Step C2: Mark inactive seats as is_available=false ─────────────────
		// GORM treats bool false as a zero value during Create, so we must do a
		// separate UPDATE after creation. Updates(map) bypasses the zero-value skip.
		markInactive := func(category string, keepCount int) {
			var toDeactivate []model.Seat
			tx.Select("id").
				Where("bus_instance_id = ? AND category = ?", instance.ID, category).
				Order("seat_number ASC").
				Offset(keepCount).
				Find(&toDeactivate)
			for _, s := range toDeactivate {
				tx.Model(&model.Seat{}).
					Where("id = ?", s.ID).
					Updates(map[string]interface{}{"is_available": false})
			}
		}
		switch seatType {
		case "seater", "semi_sleeper":
			markInactive("WOMEN", cfg.womenActive)
			markInactive("MEN", cfg.menActive)
			markInactive("GENERAL", cfg.generalActive)
		case "sleeper":
			markInactive("GENERAL", cfg.generalActive)
		}

		// ── Step D: Update BusInstance availability count ──────────────────────
		availUpdate := map[string]interface{}{}
		switch seatType {
		case "seater":
			availUpdate["available_seater"] = cfg.activeSeats()
		case "semi_sleeper":
			availUpdate["available_semi_sleeper"] = cfg.activeSeats()
		case "sleeper":
			availUpdate["available_sleeper"] = cfg.activeSeats()
		}
		if err := tx.Model(&instance).Updates(availUpdate).Error; err != nil {
			return err
		}

		// ── Step E: FareType(s) with selling price ─────────────────────────────
		// Sleeper: GENERAL (cost + 30%) + FLEXI (cost + 30% + 300)
		// Seater / Semi-Sleeper: one GENERAL fare only
		if seatType == "sleeper" {
			generalSellingPrice := sellingPrice
			flexiSellingPrice := sellingPrice + 300.0
			sleeperFares := []model.FareType{
				{
					BusInstanceID:   instance.ID,
					SeatType:        "sleeper",
					Name:            "GENERAL",
					Price:           generalSellingPrice,
					IsRefundable:    true,
					CancellationFee: 50.0,
					SeatsAvailable:  cfg.activeSeats(),
				},
				{
					BusInstanceID:   instance.ID,
					SeatType:        "sleeper",
					Name:            "FLEXI",
					Price:           flexiSellingPrice,
					IsRefundable:    true,
					CancellationFee: 100.0,
					SeatsAvailable:  cfg.activeSeats(),
				},
			}
			if err := tx.Create(&sleeperFares).Error; err != nil {
				log.Printf("[CRON ERROR] Failed to create sleeper fares for bus %s: %v\n", bus.BusNumber, err)
				return err
			}
		} else {
			fare := model.FareType{
				BusInstanceID:   instance.ID,
				SeatType:        seatType,
				Name:            "GENERAL",
				Price:           sellingPrice,
				IsRefundable:    true,
				CancellationFee: cancellationFeeFor(seatType),
				SeatsAvailable:  cfg.activeSeats(),
			}
			if err := tx.Create(&fare).Error; err != nil {
				log.Printf("[CRON ERROR] Failed to create fare for bus %s: %v\n", bus.BusNumber, err)
				return err
			}
		}

		// ── Step F: Clone boarding/dropping points ─────────────────────────────
		var templateInst model.BusInstance
		if err := tx.Joins("JOIN boarding_points ON boarding_points.bus_instance_id = bus_instances.id").
			Where("bus_id = ?", bus.ID).Order("travel_date DESC").First(&templateInst).Error; err == nil {

			var bps []model.BoardingPoint
			if err := tx.Where("bus_instance_id = ?", templateInst.ID).Find(&bps).Error; err == nil {
				for _, bp := range bps {
					offset := bp.PickupTime.Sub(templateInst.DepartureAt)
					tx.Create(&model.BoardingPoint{
						BusInstanceID: instance.ID,
						BusStopID:     bp.BusStopID,
						PickupTime:    instance.DepartureAt.Add(offset),
						Landmark:      bp.Landmark,
						SequenceOrder: bp.SequenceOrder,
					})
				}
			}
			var dps []model.DroppingPoint
			if err := tx.Where("bus_instance_id = ?", templateInst.ID).Find(&dps).Error; err == nil {
				for _, dp := range dps {
					offset := dp.DropTime.Sub(templateInst.DepartureAt)
					tx.Create(&model.DroppingPoint{
						BusInstanceID: instance.ID,
						BusStopID:     dp.BusStopID,
						DropTime:      instance.DepartureAt.Add(offset),
						Landmark:      dp.Landmark,
						SequenceOrder: dp.SequenceOrder,
					})
				}
			}
		} else {
			// Fallback: direct origin → destination points
			tx.Create(&model.BoardingPoint{
				BusInstanceID: instance.ID,
				BusStopID:     bus.OriginStopID,
				PickupTime:    instance.DepartureAt,
				SequenceOrder: 1,
				Landmark:      "Main Terminal",
			})
			tx.Create(&model.DroppingPoint{
				BusInstanceID: instance.ID,
				BusStopID:     bus.DestinationStopID,
				DropTime:      instance.ArrivalAt,
				SequenceOrder: 2,
				Landmark:      "Bus Station",
			})
		}

		log.Printf("[CRON SUCCESS] %s on %s → %d seats active, spend ₹%.2f\n",
			bus.BusNumber, targetDate.Format("2006-01-02"),
			cfg.activeSeats(), float64(cfg.activeSeats())*cfg.basePrice)
		return nil
	})

	return err == nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Seat Generation
// ─────────────────────────────────────────────────────────────────────────────

// buildSeats generates ALL physical seats for an instance.
//
// Category is FIXED by position (WOMEN → MEN → GENERAL).
// is_available is DYNAMIC: only purchased (active) seats are true.
// Inactive seats still exist in the DB and appear in the frontend layout but cannot be booked.
func buildSeats(instanceID uuid.UUID, cfg prebookingConfig) []model.Seat {
	type block struct {
		category string
		total    int
		active   int
	}

	var blocks []block
	switch cfg.seatType {
	case "seater", "semi_sleeper":
		blocks = []block{
			{"WOMEN", cfg.womenTotal, cfg.womenActive},
			{"MEN", cfg.menTotal, cfg.menActive},
			{"GENERAL", cfg.generalTotal, cfg.generalActive},
		}
	case "sleeper":
		blocks = []block{
			{"GENERAL", cfg.generalTotal, cfg.generalActive},
		}
	}

	var seats []model.Seat
	globalIdx := 0 // sequential index across all blocks (used for seat numbering)

	for _, b := range blocks {
		for i := 0; i < b.total; i++ {
			seatNum, berthType, position := seatAttributes(cfg.seatType, globalIdx)

			seats = append(seats, model.Seat{
				BusInstanceID: instanceID,
				SeatNumber:    seatNum,
				SeatType:      cfg.seatType,
				BerthType:     berthType,
				Position:      position,
				Category:      b.category,
				// IsAvailable is NOT set here intentionally.
				// DB default=true means all seats start as available.
				// A post-creation step marks inactive seats as false.
			})
			globalIdx++
		}
	}
	return seats
}

// seatAttributes computes the seat number, berth type, and position for a given global seat index.
//
// Seater layout   : 8 rows × 5 cols (2 left + 3 right) → S{row}{col}  e.g. S1A … S8E
// Semi-Sleeper    : 8 rows × 4 cols (2 left + 2 right) → SS{row}{col} e.g. SS1A … SS8D
// Sleeper         : 4 rows × 2 sides × 2 berths (L/U)  → L{row}{side}/U{row}{side} e.g. L1A, U1A
func seatAttributes(seatType string, idx int) (seatNum, berthType, position string) {
	// berthType is only meaningful for sleeper buses.
	// Seater and semi-sleeper have no berth concept → empty string.
	berthType = ""

	switch seatType {
	case "seater":
		cols := 5 // 2 left + 3 right
		row := (idx / cols) + 1
		col := idx % cols
		seatNum = fmt.Sprintf("S%d%s", row, string(rune('A'+col)))
		switch col {
		case 0, 4:
			position = "WINDOW"
		case 1, 2:
			position = "AISLE"
		default:
			position = "MIDDLE"
		}

	case "semi_sleeper":
		cols := 4 // 2 left + 2 right
		row := (idx / cols) + 1
		col := idx % cols
		seatNum = fmt.Sprintf("SS%d%s", row, string(rune('A'+col)))
		if col == 0 || col == 3 {
			position = "WINDOW"
		} else {
			position = "AISLE"
		}

	case "sleeper":
		// Pattern per row: L-A(idx%4==0), U-A(idx%4==1), L-B(idx%4==2), U-B(idx%4==3)
		row := (idx / 4) + 1
		posInRow := idx % 4
		side := "A"
		if posInRow >= 2 {
			side = "B"
		}
		if posInRow%2 == 0 {
			seatNum = fmt.Sprintf("L%d%s", row, side)
			berthType = "LOWER"
		} else {
			seatNum = fmt.Sprintf("U%d%s", row, side)
			berthType = "UPPER"
		}
		position = "WINDOW"
	}
	return
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func cancellationFeeFor(seatType string) float64 {
	switch seatType {
	case "sleeper":
		return 50.0
	case "semi_sleeper":
		return 40.0
	default:
		return 25.0
	}
}

// combineDateAndTime merges a calendar date with a time string (multiple formats).
func combineDateAndTime(busNumber string, d time.Time, timeStr string) (time.Time, error) {
	var t time.Time
	var err error

	t, err = time.Parse("2006-01-02T15:04:05Z", timeStr)
	if err != nil {
		t, err = time.Parse("15:04:05", timeStr)
		if err != nil {
			t, err = time.Parse("15:04", timeStr)
			if err != nil {
				log.Printf("[CRON ERROR] Failed to parse time '%s' for bus %s: %v\n", timeStr, busNumber, err)
				return time.Time{}, err
			}
		}
	}

	return time.Date(d.Year(), d.Month(), d.Day(), t.Hour(), t.Minute(), t.Second(), 0, d.Location()), nil
}
