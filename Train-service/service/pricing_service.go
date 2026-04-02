package service

import (
	"encoding/json"
	"log"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/config"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/gorm"
)

// RunPricingEngine runs every N minutes (set by PRICING_ENGINE_INTERVAL_MINUTES).
// Reads active pricing_rules and recalculates train_inventory.price on upcoming schedules.
func RunPricingEngine(db *gorm.DB, cfg *config.Config) {
	interval := time.Duration(cfg.PricingEngineIntervalMins) * time.Minute
	log.Printf("[pricing-engine] Started — running every %v", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on startup
	runPricingPass(db)

	for range ticker.C {
		runPricingPass(db)
	}
}

func runPricingPass(db *gorm.DB) {
	log.Println("[pricing-engine] Running pricing recalculation...")

	// Fetch all active rules ordered by priority
	var rules []models.PricingRule
	if err := db.Where("is_active = true").Order("priority ASC").Find(&rules).Error; err != nil {
		log.Printf("[pricing-engine] Failed to fetch rules: %v", err)
		return
	}

	// Fetch upcoming schedules (next 30 days)
	// We Preload Train.Stops so we can calculate the price based on the number of stops
	cutoff := time.Now().AddDate(0, 0, 30)
	var schedules []models.TrainSchedule
	if err := db.Preload("Train.Stops").
		Where("departure_at > ? AND departure_at < ? AND status != 'CANCELLED'", time.Now(), cutoff).
		Find(&schedules).Error; err != nil {
		log.Printf("[pricing-engine] Failed to fetch schedules: %v", err)
		return
	}

	updated := 0
	for _, schedule := range schedules {
		// Fetch all inventory for this schedule
		var inventory []models.TrainInventory
		if err := db.Where("train_schedule_id = ? AND status = 'AVAILABLE'", schedule.ID).
			Find(&inventory).Error; err != nil {
			continue
		}
		if len(inventory) == 0 {
			continue
		}

		// Compute total capacity per class for fill-rate calculation
		totalByClass := map[string]int{}
		soldByClass := map[string]int{}
		for _, item := range inventory {
			totalByClass[item.Class]++
			if item.Status == "BOOKED" {
				soldByClass[item.Class]++
			}
		}

		// Calculate total stops for this train to factor into the price
		stopCount := len(schedule.Train.Stops)
		// Assume a fixed addition per stop. You can modify this value (e.g., 20.0 per stop)
		pricePerStop := 20.0
		stopsPriceAdded := float64(stopCount) * pricePerStop

		for _, item := range inventory {
			// 1. BASE PRICE + STOPS PRICE
			basePrice := 100.0 + stopsPriceAdded

			// 2. CLASS MULTIPLIER
			classMultiplier := 1.0
			switch item.Class {
			case "SL":
				classMultiplier = 1.0 // Standard
			case "3AC":
				classMultiplier = 2.5 // 2.5x of Sleeper
			case "2AC":
				classMultiplier = 4.0 // 4x of Sleeper
			case "1AC":
				classMultiplier = 6.0 // 6x of Sleeper
			}

			// 3. BERTH MULTIPLIER
			berthMultiplier := 1.0
			switch item.BerthType {
			case "LOWER", "SIDE_LOWER":
				berthMultiplier = 1.10 // 10% Premium for lower berths
			case "MIDDLE", "UPPER", "SIDE_UPPER":
				berthMultiplier = 1.00 // Standard
			}

			// 4. DYNAMIC PRICING MULTIPLIER (Demand, Time to Departure, Seasonal rules)
			dynamicMultiplier := applyRules(rules, item, schedule, totalByClass, soldByClass)

			// 5. FINAL CALCULATION
			finalPrice := basePrice * classMultiplier * berthMultiplier * dynamicMultiplier

			// Round to 2 decimal places
			finalPrice = float64(int(finalPrice*100+0.5)) / 100

			// Update price in DB
			if err := db.Model(&models.TrainInventory{}).
				Where("id = ?", item.ID).
				Update("price", finalPrice).Error; err != nil {
				log.Printf("[pricing-engine] Failed to update price for seat %s: %v", item.ID, err)
			} else {
				updated++
			}
		}
	}
	log.Printf("[pricing-engine] Updated %d seat prices", updated)
}

// applyRules applies all matching pricing rules and stacks their multipliers.
func applyRules(rules []models.PricingRule, item models.TrainInventory, schedule models.TrainSchedule,
	totalByClass, soldByClass map[string]int) float64 {

	multiplier := 1.0
	if len(rules) == 0 {
		return multiplier
	}

	daysBeforeDep := time.Until(schedule.DepartureAt).Hours() / 24

	for _, rule := range rules {
		var conditions map[string]interface{}
		if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
			continue
		}

		switch rule.RuleType {
		case "DEMAND":
			total := totalByClass[item.Class]
			sold := soldByClass[item.Class]
			if total > 0 {
				fillRate := float64(sold) / float64(total)
				if threshold, ok := conditions["fill_rate_above"].(float64); ok {
					if fillRate > threshold {
						multiplier *= float64(rule.Multiplier)
					}
				}
			}
		case "TIME_TO_DEPARTURE":
			if above, ok := conditions["days_before_above"].(float64); ok {
				if daysBeforeDep > above {
					multiplier *= float64(rule.Multiplier)
				}
			}
			if below, ok := conditions["days_before_below"].(float64); ok {
				if daysBeforeDep < below {
					multiplier *= float64(rule.Multiplier)
				}
			}
		case "SEASONAL":
			if months, ok := conditions["months"].([]interface{}); ok {
				currentMonth := int(time.Now().Month())
				for _, m := range months {
					if int(m.(float64)) == currentMonth {
						multiplier *= float64(rule.Multiplier)
						break
					}
				}
			}
		}
	}
	return multiplier
}
