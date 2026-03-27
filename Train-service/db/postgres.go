package db

import (
	"encoding/json"
	"log"

	"github.com/nabeel-mp/tripneo/train-service/config"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectPostgres(cfg *config.Config) {
	db, err := gorm.Open(postgres.Open(cfg.DB_URL), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	log.Println("Connected to PostgreSQL (train-service)")

	err = db.AutoMigrate(
		&models.Station{},
		&models.Train{},
		&models.TrainStop{},
		&models.TrainSchedule{},
		&models.TrainInventory{},
		&models.TrainBooking{},
		&models.BookingSeat{},
		&models.Passenger{},
		&models.TrainTicket{},
		&models.CancellationPolicy{},
		&models.Cancellation{},
		&models.PricingRule{},
	)

	if err != nil {
		log.Fatal("AutoMigrate failed:", err)
	}
	log.Println("AutoMigrate complete")

	constraints := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_train_schedule_unique
			ON train_schedules(train_id, schedule_date)`,

		`CREATE UNIQUE INDEX IF NOT EXISTS idx_inventory_seat_unique
			ON train_inventory(train_schedule_id, coach, seat_number)`,
	}

	for _, sql := range constraints {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("Warning: constraint may already exist: %v", err)
		}
	}
	log.Println("Constraints applied")

	seedCancellationPolicies(db)
	seedPricingRules(db)

	DB = db
}

func seedCancellationPolicies(db *gorm.DB) {
	var count int64
	db.Model(&models.CancellationPolicy{}).Count(&count)
	if count > 0 {
		log.Println("Cancellation policies already seeded — skipping")
		return
	}

	policies := []models.CancellationPolicy{
		{
			Name:                 "Full Refund (7+ days before)",
			HoursBeforeDeparture: 168,
			RefundPercentage:     90.00,
			CancellationFee:      0,
			IsActive:             true,
		},
		{
			Name:                 "Partial Refund (2-7 days before)",
			HoursBeforeDeparture: 48,
			RefundPercentage:     50.00,
			CancellationFee:      0,
			IsActive:             true,
		},
		{
			Name:                 "No Refund (under 48 hours)",
			HoursBeforeDeparture: 0,
			RefundPercentage:     0.00,
			CancellationFee:      0,
			IsActive:             true,
		},
	}

	if err := db.Create(&policies).Error; err != nil {
		log.Printf("Warning: failed to seed cancellation policies: %v", err)
		return
	}
	log.Println("Cancellation policies seeded (3 rows)")
}

// seedPricingRules seeds the 6 default dynamic pricing rules.
func seedPricingRules(db *gorm.DB) {
	var count int64
	db.Model(&models.PricingRule{}).Count(&count)
	if count > 0 {
		log.Println("Pricing rules already seeded — skipping")
		return
	}

	marshal := func(v interface{}) []byte {
		b, _ := json.Marshal(v)
		return b
	}

	rules := []models.PricingRule{
		{
			Name:       "Demand Medium",
			RuleType:   "DEMAND",
			Conditions: marshal(map[string]interface{}{"fill_rate_above": 0.70}),
			Multiplier: 1.15,
			Priority:   10,
			IsActive:   true,
		},
		{
			Name:       "Demand High",
			RuleType:   "DEMAND",
			Conditions: marshal(map[string]interface{}{"fill_rate_above": 0.90}),
			Multiplier: 1.35,
			Priority:   20,
			IsActive:   true,
		},
		{
			Name:       "Early Bird",
			RuleType:   "TIME_TO_DEPARTURE",
			Conditions: marshal(map[string]interface{}{"days_before_above": 30}),
			Multiplier: 0.90,
			Priority:   5,
			IsActive:   true,
		},
		{
			Name:       "Last Minute",
			RuleType:   "TIME_TO_DEPARTURE",
			Conditions: marshal(map[string]interface{}{"days_before_below": 3}),
			Multiplier: 1.25,
			Priority:   15,
			IsActive:   true,
		},
		{
			Name:       "Peak Season",
			RuleType:   "SEASONAL",
			Conditions: marshal(map[string]interface{}{"months": []int{12, 1, 4, 5}}),
			Multiplier: 1.20,
			Priority:   8,
			IsActive:   true,
		},
		{
			Name:       "Off Peak",
			RuleType:   "SEASONAL",
			Conditions: marshal(map[string]interface{}{"months": []int{6, 7, 8}}),
			Multiplier: 0.85,
			Priority:   7,
			IsActive:   true,
		},
	}

	if err := db.Create(&rules).Error; err != nil {
		log.Printf("Warning: failed to seed pricing rules: %v", err)
		return
	}
	log.Println("Pricing rules seeded (6 rows)")
}
