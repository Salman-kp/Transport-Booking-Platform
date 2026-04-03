package service

import (
	"context"
	"log"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/kafka"
	"github.com/nabeel-mp/tripneo/train-service/repository"
	"github.com/nabeel-mp/tripneo/train-service/utils"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RunExpiryWorker runs every 5 minutes, finds PENDING_PAYMENT bookings past their
// expires_at timestamp, marks them EXPIRED, releases Redis seat locks, restores
// seat availability, and publishes train.booking.expired Kafka events.
func RunExpiryWorker(db *gorm.DB, rdb *goredis.Client, producer *kafka.Producer) {
	log.Println("[expiry-worker] Started — checking every 5 minutes")
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Run immediately on startup
	runExpiryPass(db, rdb, producer)

	for range ticker.C {
		runExpiryPass(db, rdb, producer)
	}
}

func runExpiryPass(db *gorm.DB, rdb *goredis.Client, producer *kafka.Producer) {
	ctx := context.Background()

	bookings, err := repository.GetExpiredPendingBookings()
	if err != nil {
		log.Printf("[expiry-worker] Failed to fetch expired bookings: %v", err)
		return
	}
	if len(bookings) == 0 {
		return
	}

	log.Printf("[expiry-worker] Found %d expired bookings to process", len(bookings))

	for _, booking := range bookings {
		seatIDs, err := repository.GetSeatIDsByBooking(booking.ID.String())
		if err != nil {
			log.Printf("[expiry-worker] Failed to get seats for booking %s: %v", booking.ID, err)
			continue
		}

		txErr := db.Transaction(func(tx *gorm.DB) error {
			// Mark booking EXPIRED
			if err := repository.UpdateBookingStatus(tx, booking.ID.String(), "EXPIRED"); err != nil {
				return err
			}
			if len(seatIDs) > 0 {
				// Restore seats to AVAILABLE
				if err := repository.UpdateSeatStatuses(tx, seatIDs, "AVAILABLE"); err != nil {
					return err
				}
				// Restore schedule availability count
				if err := repository.IncrementAvailability(tx, booking.ScheduleID.String(), booking.SeatClass, len(seatIDs)); err != nil {
					return err
				}
			}
			return nil
		})

		if txErr != nil {
			log.Printf("[expiry-worker] Failed to expire booking %s: %v", booking.ID, txErr)
			continue
		}

		// Release Redis locks (safe even if already expired)
		if len(seatIDs) > 0 {
			_ = utils.UnlockSeats(ctx, rdb, booking.ScheduleID.String(), seatIDs)
		}

		// Publish expired event
		producer.PublishBookingExpired(ctx, kafka.BookingExpiredEvent{
			BookingID: booking.ID.String(),
			PNR:       booking.PNR,
			UserID:    booking.UserID,
		})

		log.Printf("[expiry-worker] Expired booking %s (PNR: %s)", booking.ID, booking.PNR)
	}
}
