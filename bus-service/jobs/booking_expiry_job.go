package jobs

import (
	"log"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

func CleanupExpiredBookings(db *gorm.DB) {
	log.Println("CRON Running stale bookings cleanup")

	now := time.Now()

	res := db.Model(&model.Booking{}).
		Where("status = ?", "PENDING_PAYMENT").
		Where("expires_at < ?", now).
		Update("status", "EXPIRED")

	if res.Error != nil {
		log.Println("[CRON ERROR] Failed to run booking cleanup:", res.Error)
		return
	}

	if res.RowsAffected > 0 {
		log.Printf("CRON Cleanup completed: marked %d stale bookings as EXPIRED\n", res.RowsAffected)
	}
}
