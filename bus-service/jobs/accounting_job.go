package jobs

import (
	"log"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

// FinalizePrebookingAccounting checks for bus instances that have departed
// and finalizes their PrebookingAccounting records by calculating the loss.
func FinalizePrebookingAccounting(db *gorm.DB) {
	log.Println("[CRON] Running prebooking accounting finalization job...")

	var accountings []model.PrebookingAccounting
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Find all accountings where departure time is TODAY, has already passed, and they are not finalized yet.
	if err := db.Where("is_finalized = ? AND departure_date_time >= ? AND departure_date_time < ? AND departure_date_time <= ?",
		false, startOfDay, endOfDay, now).Find(&accountings).Error; err != nil {
		log.Println("[CRON ERROR] Failed to fetch prebooking accountings:", err)
		return
	}

	for _, acc := range accountings {
		loss := acc.SpendAmountTotal - acc.ProfitAmount
		if loss < 0 {
			loss = 0
		}

		if err := db.Model(&acc).Updates(map[string]interface{}{
			"loss_amount":  loss,
			"is_finalized": true,
			"updated_at":   time.Now(),
		}).Error; err != nil {
			log.Printf("[CRON ERROR] Failed to finalize accounting for instance %s: %v\n", acc.InstanceID, err)
		} else {
			log.Printf("[CRON SUCCESS] Finalized accounting for instance %s with loss %.2f\n", acc.InstanceID, loss)
		}
	}

	log.Println("[CRON] Prebooking accounting finalization job completed.")
}
