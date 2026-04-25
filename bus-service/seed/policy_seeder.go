package seed

import (
	"encoding/json"
	"log"
	"os"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

func SeedCancellationPolicies(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/cancellation_policies.json")
	if err != nil {
		return err
	}
	var records []model.CancellationPolicy
	if err := json.Unmarshal(bytes, &records); err != nil {
		return err
	}

	for _, r := range records {
		// 1. Validation
		if r.Name == "" || r.HoursBeforeDeparture < 0 || r.RefundPercentage < 0 || r.RefundPercentage > 100 {
			log.Printf("[seed] skipping invalid cancellation policy: %+v\n", r.Name)
			continue
		}

		// 2. Uniqueness & Controlled Update
		var existing model.CancellationPolicy
		err := tx.Where("name = ? AND hours_before_departure = ?", r.Name, r.HoursBeforeDeparture).First(&existing).Error
		if err == nil {
			// Update if exists (Production-grade update logic)
			existing.RefundPercentage = r.RefundPercentage
			existing.CancellationFee = r.CancellationFee
			existing.IsActive = r.IsActive
			if err := tx.Save(&existing).Error; err != nil {
				log.Printf("[seed] failed to update policy %s: %v\n", r.Name, err)
				return err
			}
		} else {
			// Create if not exists
			if err := tx.Create(&r).Error; err != nil {
				log.Printf("[seed] failed to create policy %s: %v\n", r.Name, err)
				return err
			}
		}
	}
	log.Println("✅ Cancellation policies seeding completed")
	return nil
}

func SeedPricingRules(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/pricing_rules.json")
	if err != nil {
		return err
	}
	var records []model.PricingRule
	if err := json.Unmarshal(bytes, &records); err != nil {
		return err
	}

	for _, r := range records {
		// 1. Validation
		if r.Name == "" || r.RuleType == "" || r.Multiplier <= 0 {
			log.Printf("[seed] skipping invalid pricing rule: %+v\n", r.Name)
			continue
		}

		// 2. Uniqueness & Controlled Update
		var existing model.PricingRule
		err := tx.Where("name = ? AND rule_type = ?", r.Name, r.RuleType).First(&existing).Error
		if err == nil {
			// Update
			existing.Conditions = r.Conditions
			existing.Multiplier = r.Multiplier
			existing.Priority = r.Priority
			existing.IsActive = r.IsActive
			if err := tx.Save(&existing).Error; err != nil {
				log.Printf("[seed] failed to update pricing rule %s: %v\n", r.Name, err)
				return err
			}
		} else {
			// Create
			if err := tx.Create(&r).Error; err != nil {
				log.Printf("[seed] failed to create pricing rule %s: %v\n", r.Name, err)
				return err
			}
		}
	}
	log.Println("✅ Pricing rules seeding completed")
	return nil
}
