package repository

import (
	"github.com/Salman-kp/tripneo/bus-service/db"
	"github.com/Salman-kp/tripneo/bus-service/model"
)

// Fetches all active pricing rules descending by priority (highest priority first)
func GetActivePricingRules() ([]model.PricingRule, error) {
	var rules []model.PricingRule
	err := db.DB.Where("is_active = true").Order("priority DESC").Find(&rules).Error
	return rules, err
}

// Overwrite a pricing rule by ID
func UpdatePricingRule(id string, rule *model.PricingRule) error {
	return db.DB.Model(&model.PricingRule{}).Where("id = ?", id).Updates(rule).Error
}

// Engine execution hook for updating the current dynamically adjusted bus-instance price
func UpdateBusInstancePrices(instanceID string, seater, semiSleeper, sleeper float64) error {
	return db.DB.Model(&model.BusInstance{}).Where("id = ?", instanceID).Updates(map[string]interface{}{
		"current_price_seater":        seater,
		"current_price_semi_sleeper":  semiSleeper,
		"current_price_sleeper":       sleeper,
	}).Error
}
