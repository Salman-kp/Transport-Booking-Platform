package seed

import (
	"encoding/json"
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
		if err := tx.Where("name = ?", r.Name).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
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
		if err := tx.Where("name = ?", r.Name).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
	return nil
}
