package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/config"
	"github.com/nabeel-mp/tripneo/train-service/repository"
	"github.com/nabeel-mp/tripneo/train-service/utils"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// StartConsumer listens to payment events. Safe to call even when Kafka is not available.
func StartConsumer(cfg *config.Config, db *gorm.DB, rdb *goredis.Client, producer *Producer) {
	if cfg.KAFKA_BROKERS == "" {
		log.Println("[kafka] No broker configured — Kafka consumer disabled")
		return
	}

	topics := []string{TopicPaymentCompleted, TopicPaymentFailed, TopicPaymentRefunded}

	for _, topic := range topics {
		go consumeTopic(cfg, db, rdb, producer, topic)
	}
}

func consumeTopic(cfg *config.Config, db *gorm.DB, rdb *goredis.Client, producer *Producer, topic string) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{cfg.KAFKA_BROKERS},
		GroupID:        cfg.KAFKA_GROUP_ID,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
	defer r.Close()

	log.Printf("[kafka] Consumer started for topic: %s", topic)

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("[kafka] Read error on %s: %v", topic, err)
			time.Sleep(5 * time.Second)
			continue
		}
		handleMessage(db, rdb, producer, topic, m.Value)
	}
}

func handleMessage(db *gorm.DB, rdb *goredis.Client, producer *Producer, topic string, value []byte) {
	ctx := context.Background()
	switch topic {

	case TopicPaymentCompleted:
		var evt PaymentCompletedEvent
		if err := json.Unmarshal(value, &evt); err != nil {
			log.Printf("[kafka] Failed to unmarshal PaymentCompleted: %v", err)
			return
		}
		handlePaymentCompleted(ctx, db, rdb, producer, evt)

	case TopicPaymentFailed:
		var evt PaymentFailedEvent
		if err := json.Unmarshal(value, &evt); err != nil {
			log.Printf("[kafka] Failed to unmarshal PaymentFailed: %v", err)
			return
		}
		handlePaymentFailed(ctx, db, rdb, evt)

	case TopicPaymentRefunded:
		var evt PaymentRefundedEvent
		if err := json.Unmarshal(value, &evt); err != nil {
			log.Printf("[kafka] Failed to unmarshal PaymentRefunded: %v", err)
			return
		}
		handlePaymentRefunded(db, evt)
	}
}

// handlePaymentCompleted: confirm booking, mark seats BOOKED, issue ticket, notify.
func handlePaymentCompleted(ctx context.Context, db *gorm.DB, rdb *goredis.Client, producer *Producer, evt PaymentCompletedEvent) {
	booking, err := repository.GetBookingByID(evt.BookingID)
	if err != nil {
		log.Printf("[kafka] PaymentCompleted: booking %s not found: %v", evt.BookingID, err)
		return
	}

	if booking.Status != "PENDING_PAYMENT" {
		log.Printf("[kafka] PaymentCompleted: booking %s already in status %s — skipping", evt.BookingID, booking.Status)
		return
	}

	// Get seat IDs for this booking
	seatIDs, err := repository.GetSeatIDsByBooking(evt.BookingID)
	if err != nil {
		log.Printf("[kafka] PaymentCompleted: failed to get seats for booking %s: %v", evt.BookingID, err)
		return
	}

	// DB transaction: confirm booking + mark seats BOOKED
	txErr := db.Transaction(func(tx *gorm.DB) error {
		if err := repository.ConfirmBooking(tx, evt.BookingID, evt.PaymentRef); err != nil {
			return err
		}
		return repository.MarkSeatsBooked(tx, seatIDs)
	})
	if txErr != nil {
		log.Printf("[kafka] PaymentCompleted: failed to confirm booking %s: %v", evt.BookingID, txErr)
		return
	}

	// Release Redis seat locks (no longer needed — booking is confirmed)
	_ = utils.UnlockSeats(ctx, rdb, booking.ScheduleID.String(), seatIDs)

	log.Printf("[kafka] Booking %s (PNR: %s) CONFIRMED", evt.BookingID, booking.PNR)

	// Publish confirmed event
	producer.PublishBookingConfirmed(ctx, BookingConfirmedEvent{
		BookingID:   booking.ID.String(),
		PNR:         booking.PNR,
		UserID:      booking.UserID,
		TotalAmount: booking.TotalAmount,
	})

	// Notify user
	producer.PublishNotification(ctx, NotificationEvent{
		Type:            "BOOKING_CONFIRMED",
		RecipientUserID: booking.UserID,
		Template:        "booking_confirmed",
		Data: map[string]interface{}{
			"pnr":          booking.PNR,
			"booking_id":   booking.ID.String(),
			"total_amount": booking.TotalAmount,
		},
	})
}

// handlePaymentFailed: mark booking FAILED, release seat locks.
func handlePaymentFailed(ctx context.Context, db *gorm.DB, rdb *goredis.Client, evt PaymentFailedEvent) {
	booking, err := repository.GetBookingByID(evt.BookingID)
	if err != nil {
		log.Printf("[kafka] PaymentFailed: booking %s not found: %v", evt.BookingID, err)
		return
	}

	seatIDs, _ := repository.GetSeatIDsByBooking(evt.BookingID)

	db.Transaction(func(tx *gorm.DB) error {
		repository.UpdateBookingStatus(tx, evt.BookingID, "FAILED")
		repository.UpdateSeatStatuses(tx, seatIDs, "AVAILABLE")
		repository.IncrementAvailability(tx, booking.ScheduleID.String(), booking.SeatClass, len(seatIDs))
		return nil
	})

	_ = utils.UnlockSeats(ctx, rdb, booking.ScheduleID.String(), seatIDs)
	log.Printf("[kafka] Booking %s marked FAILED", evt.BookingID)
}

// handlePaymentRefunded: mark cancellation refund_status = COMPLETED.
func handlePaymentRefunded(db *gorm.DB, evt PaymentRefundedEvent) {
	now := time.Now()
	db.Exec(
		"UPDATE cancellations SET refund_status = ?, processed_at = ? WHERE booking_id = ?",
		"COMPLETED", now, evt.BookingID,
	)
	log.Printf("[kafka] Refund COMPLETED for booking %s", evt.BookingID)
}
