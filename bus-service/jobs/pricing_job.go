package jobs

import (
	"encoding/json"
	"log"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

type Conditions struct {
	FillRateAbove             float64 `json:"fill_rate_above,omitempty"`
	HoursBeforeDepartureAbove float64 `json:"hours_before_departure_above,omitempty"`
	HoursBeforeDepartureBelow float64 `json:"hours_before_departure_below,omitempty"`
	Days                      []int   `json:"days,omitempty"`
}

func UpdatePricesBasedOnRules(db *gorm.DB) {
	log.Println("[CRON] Running dynamic pricing update job...")

	var rules []model.PricingRule
	if err := db.Where("is_active = ?", true).Order("priority asc").Find(&rules).Error; err != nil {
		log.Println("[CRON ERROR] Failed to fetch pricing rules:", err)
		return
	}

	var instances []model.BusInstance
	// Only update scheduled trips that haven't departed yet
	if err := db.Where("status = ? AND departure_at > ?", "SCHEDULED", time.Now()).Find(&instances).Error; err != nil {
		log.Println("[CRON ERROR] Failed to fetch bus instances:", err)
		return
	}

	for _, instance := range instances {
		applyRulesToInstance(db, instance, rules)
	}

	log.Println("[CRON] Dynamic pricing update completed.")
}

func applyRulesToInstance(db *gorm.DB, instance model.BusInstance, rules []model.PricingRule) {
	now := time.Now()
	hoursUntilDeparture := instance.DepartureAt.Sub(now).Hours()

	weekday := int(instance.DepartureAt.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	// Calculate fill rate
	var totalSeats int64
	var bookedSeats int64
	db.Model(&model.Seat{}).Where("bus_instance_id = ?", instance.ID).Count(&totalSeats)
	db.Model(&model.Seat{}).Where("bus_instance_id = ? AND is_available = ?", instance.ID, false).Count(&bookedSeats)

	fillRate := 0.0
	if totalSeats > 0 {
		fillRate = float64(bookedSeats) / float64(totalSeats)
	}

	// Determine applicable rules
	typeMultiplier := make(map[string]float64)

	for _, rule := range rules {
		if _, exists := typeMultiplier[rule.RuleType]; exists {
			continue
		}

		var cond Conditions
		if err := json.Unmarshal(rule.Conditions, &cond); err != nil {
			continue
		}

		matched := false
		switch rule.RuleType {
		case "DEMAND":
			if cond.FillRateAbove > 0 && fillRate >= cond.FillRateAbove {
				matched = true
			}
		case "TIME_TO_DEPARTURE":
			if cond.HoursBeforeDepartureAbove > 0 && hoursUntilDeparture >= cond.HoursBeforeDepartureAbove {
				matched = true
			} else if cond.HoursBeforeDepartureBelow > 0 && hoursUntilDeparture <= cond.HoursBeforeDepartureBelow {
				matched = true
			}
		case "WEEKEND":
			for _, d := range cond.Days {
				if d == weekday {
					matched = true
					break
				}
			}
		}

		if matched {
			typeMultiplier[rule.RuleType] = rule.Multiplier
		}
	}

	finalMultiplier := 1.0
	for _, m := range typeMultiplier {
		finalMultiplier *= m
	}

	// Update FareTypes
	var fares []model.FareType
	if err := db.Where("bus_instance_id = ?", instance.ID).Find(&fares).Error; err != nil {
		log.Printf("[CRON ERROR] Failed to fetch fares for instance %s: %v\n", instance.ID, err)
		return
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, fare := range fares {
			basePrice := 0.0
			switch fare.SeatType {
			case "seater":
				basePrice = instance.BasePriceSeater
			case "semi_sleeper":
				basePrice = instance.BasePriceSemiSleeper
			case "sleeper":
				basePrice = instance.BasePriceSleeper
			}

			if basePrice > 0 {
				offset := 0.0
				if fare.Name == "FLEXI" || fare.Name == "SUPER FLEXI" {
					offset = 300.0
				}

				newPrice := (basePrice * finalMultiplier) + offset
				if err := tx.Model(&fare).Update("price", newPrice).Error; err != nil {
					return err
				}
			}
		}

		// Also update the CurrentPrice fields in BusInstance for search fast-lookup
		// ONLY update fields where the base price is > 0
		updates := map[string]interface{}{
			"updated_at": time.Now(),
		}
		if instance.BasePriceSeater > 0 {
			updates["current_price_seater"] = instance.BasePriceSeater * finalMultiplier
		}
		if instance.BasePriceSemiSleeper > 0 {
			updates["current_price_semi_sleeper"] = instance.BasePriceSemiSleeper * finalMultiplier
		}
		if instance.BasePriceSleeper > 0 {
			updates["current_price_sleeper"] = instance.BasePriceSleeper * finalMultiplier
		}

		if err := tx.Model(&instance).Updates(updates).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("[CRON ERROR] Failed to update prices for instance %s: %v\n", instance.ID, err)
	} else {
		if finalMultiplier != 1.0 {
			log.Printf("[CRON SUCCESS] Updated prices for instance %s with multiplier %.2f\n", instance.ID, finalMultiplier)
		}
	}
}
