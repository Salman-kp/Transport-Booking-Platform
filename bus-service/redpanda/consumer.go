package redpanda

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/config"
	"github.com/Salman-kp/tripneo/bus-service/model"
	busredis "github.com/Salman-kp/tripneo/bus-service/redis"
	"github.com/Salman-kp/tripneo/bus-service/repository"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// StartConsumer listens to payment events synchronously.
func StartConsumer(cfg *config.Config, db *gorm.DB, rdb *goredis.Client, producer *Producer, repo repository.BookingRepository) {
	if cfg.REDPANDA_BROKERS == "" {
		log.Println("[bus-redpanda] No broker configured — Redpanda consumer disabled")
		return
	}

	topics := []string{TopicPaymentCompleted, TopicPaymentFailed, TopicPaymentRefunded}

	for _, topic := range topics {
		go consumeTopic(cfg, db, rdb, producer, repo, topic)
	}
}

func consumeTopic(cfg *config.Config, db *gorm.DB, rdb *goredis.Client, producer *Producer, repo repository.BookingRepository, topic string) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{cfg.REDPANDA_BROKERS},
		GroupID:        cfg.REDPANDA_GROUP_ID,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})
	defer r.Close()

	log.Printf("[bus-redpanda] Consumer started for topic: %s", topic)

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("[bus-redpanda] Read error on %s: %v", topic, err)
			time.Sleep(5 * time.Second)
			continue
		}
		handleMessage(db, rdb, producer, repo, topic, m.Value)
	}
}

func handleMessage(db *gorm.DB, rdb *goredis.Client, producer *Producer, repo repository.BookingRepository, topic string, value []byte) {
	log.Printf("received event %s", topic)
	ctx := context.Background()
	switch topic {

	case TopicPaymentCompleted:
		var evt PaymentCompletedEvent
		if err := json.Unmarshal(value, &evt); err != nil {
			log.Printf("[bus-redpanda] Failed to unmarshal PaymentCompleted: %v", err)
			return
		}
		handlePaymentCompleted(ctx, db, rdb, producer, repo, evt)

	case TopicPaymentFailed:
		var evt PaymentFailedEvent
		if err := json.Unmarshal(value, &evt); err != nil {
			log.Printf("[bus-redpanda] Failed to unmarshal PaymentFailed: %v", err)
			return
		}
		handlePaymentFailed(ctx, db, rdb, evt)

	case TopicPaymentRefunded:
		var evt PaymentRefundedEvent
		if err := json.Unmarshal(value, &evt); err != nil {
			log.Printf("[bus-redpanda] Failed to unmarshal PaymentRefunded: %v", err)
			return
		}
		handlePaymentRefunded(db, evt)
	}
}

// handlePaymentCompleted delegates logic safely reproducing constraints using raw robust operations strictly explicitly.
func handlePaymentCompleted(ctx context.Context, db *gorm.DB, rdb *goredis.Client, producer *Producer, repo repository.BookingRepository, evt PaymentCompletedEvent) {
	var booking model.Booking
	if err := db.Preload("Passengers").Preload("BusInstance.Bus.Operator").
		Preload("BusInstance.Bus.OriginStop").Preload("BusInstance.Bus.DestinationStop").
		Preload("BoardingPoint.BusStop").Preload("DroppingPoint.BusStop").
		Where("id = ?", evt.BookingID).First(&booking).Error; err != nil {
		log.Printf("[bus-redpanda] PaymentCompleted: booking %s not found: %v", evt.BookingID, err)
		return
	}

	if booking.Status != "PENDING_PAYMENT" {
		log.Printf("[bus-redpanda] PaymentCompleted: booking %s already in status %s — skipping", evt.BookingID, booking.Status)
		return
	}

	if booking.ExpiresAt != nil && time.Now().After(*booking.ExpiresAt) {
		repo.UpdateBookingStatus(booking.ID.String(), booking.UserID.String(), "EXPIRED", "")
		log.Printf("[bus-redpanda] PaymentCompleted: booking %s has organically EXPIRED — rejecting confirmation safely", evt.BookingID)
		return
	}

	// Persist the payment ref safely using the structured core
	if err := repo.UpdateBookingStatus(booking.ID.String(), booking.UserID.String(), "CONFIRMED", evt.PaymentRef); err != nil {
		log.Printf("[bus-redpanda] PaymentCompleted: DB logic failed marking order %v", err)
		return
	}

	seatIDs := make([]string, 0, len(booking.Passengers))
	for _, p := range booking.Passengers {
		if p.SeatID != nil {
			seatIDs = append(seatIDs, p.SeatID.String())
		}
	}

	if len(seatIDs) > 0 {
		_ = repo.UpdateMultipleSeatsAvailability(seatIDs, false)
		_ = busredis.UnlockSeatsByOwner(ctx, rdb, booking.BusInstanceID.String(), seatIDs, booking.UserID.String())
		_ = repo.DecrementInventoryOnConfirm(booking.BusInstanceID.String(), booking.FareTypeID.String(), booking.SeatType, len(seatIDs))
	}

	ticketNumber := "BUS-" + booking.PNR
	eTicket := &model.ETicket{
		BookingID:    booking.ID,
		TicketNumber: ticketNumber,
		QRCodeURL:    "https://storage.tripneo.com/qr/bus/" + booking.PNR + ".png",
		QRData:       "SIGNED:" + booking.PNR + ":" + booking.ID.String(),
	}
	_ = repo.SaveETicket(eTicket)

	producer.PublishBookingConfirmed(ctx, BookingConfirmedEvent{
		BookingID:      booking.ID.String(),
		PNR:            booking.PNR,
		UserID:         booking.UserID.String(),
		BusNumber:      booking.BusInstance.Bus.BusNumber,
		Operator:       booking.BusInstance.Bus.Operator.Name,
		Origin:         booking.BusInstance.Bus.OriginStop.City,
		Destination:    booking.BusInstance.Bus.DestinationStop.City,
		DepartureAt:    booking.BusInstance.DepartureAt.Format(time.RFC3339),
		BoardingPoint:  booking.BoardingPoint.BusStop.Name,
		DroppingPoint:  booking.DroppingPoint.BusStop.Name,
		TotalAmount:    booking.TotalAmount,
		TicketNumber:   ticketNumber,
		PassengerCount: len(seatIDs),
	})

	producer.PublishNotification(ctx, NotificationEvent{
		Type:            "BUS_BOOKING_CONFIRMED",
		RecipientUserID: booking.UserID.String(),
		Template:        "bus_ticket_confirmed",
		Data: map[string]interface{}{
			"pnr":           booking.PNR,
			"ticket_number": ticketNumber,
			"bus_number":    booking.BusInstance.Bus.BusNumber,
			"departure":     booking.BusInstance.DepartureAt.Format(time.RFC822),
		},
	})

	log.Printf("[bus-redpanda] Booking %s (PNR: %s) successfully confirmed via Payment Event", booking.ID, booking.PNR)
}

// handlePaymentFailed marks the booking as FAILED and conditionally removes any bound resources dynamically natively.
func handlePaymentFailed(ctx context.Context, db *gorm.DB, rdb *goredis.Client, evt PaymentFailedEvent) {
	var booking model.Booking
	if err := db.Preload("Passengers").Where("id = ?", evt.BookingID).First(&booking).Error; err != nil {
		log.Printf("[bus-redpanda] PaymentFailed: booking %s not found: %v", evt.BookingID, err)
		return
	}

	if booking.Status != "PENDING_PAYMENT" {
		log.Printf("[bus-redpanda] PaymentFailed: booking %s already %s — ignoring failure", evt.BookingID, booking.Status)
		return
	}

	db.Transaction(func(tx *gorm.DB) error {
		tx.Model(&model.Booking{}).Where("id = ?", evt.BookingID).Update("status", "FAILED")
		return nil
	})

	// Release locks aggressively
	seatIDs := make([]string, 0, len(booking.Passengers))
	for _, p := range booking.Passengers {
		if p.SeatID != nil {
			seatIDs = append(seatIDs, p.SeatID.String())
		}
	}
	_ = busredis.UnlockSeatsByOwner(ctx, rdb, booking.BusInstanceID.String(), seatIDs, booking.UserID.String())

	log.Printf("[bus-redpanda] Booking %s marked FAILED due to Payment Event", evt.BookingID)
}

// handlePaymentRefunded matches cancellation constraints updating internal refund allocations natively.
func handlePaymentRefunded(db *gorm.DB, evt PaymentRefundedEvent) {
	now := time.Now()
	res := db.Exec(
		"UPDATE cancellations SET refund_status = 'COMPLETED', processed_at = ? WHERE booking_id = ?",
		now, evt.BookingID,
	)
	if res.RowsAffected > 0 {
		log.Printf("[bus-redpanda] Refund successfully mapped COMPLETED for booking %s", evt.BookingID)
	} else {
		log.Printf("[bus-redpanda] Refund received but no active Cancellation record found for booking %s", evt.BookingID)
	}
}
