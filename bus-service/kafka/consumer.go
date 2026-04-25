package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type PaymentCompletedEvent struct {
	BookingID string  `json:"booking_id"`
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	UserID    string  `json:"user_id"`
	Status    string  `json:"status"`
}

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(broker, topic, groupID string) *Consumer {
	if broker == "" {
		log.Println("[kafka] No broker configured — Kafka consumer disabled")
		return nil
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10,
		MaxBytes: 10e6,
	})
	log.Printf("[kafka] Consumer listening on topic %s", topic)
	return &Consumer{reader: r}
}

func (c *Consumer) ConsumePaymentEvents(ctx context.Context, handler func(PaymentCompletedEvent)) {
	if c == nil || c.reader == nil {
		return
	}
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("[kafka] Error reading message: %v", err)
			break
		}

		var evt PaymentCompletedEvent
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			log.Printf("[kafka] Error unmarshaling event: %v", err)
			continue
		}

		log.Printf("[kafka] Received payment event for booking: %s", evt.BookingID)
		handler(evt)
	}
}

func (c *Consumer) Close() {
	if c != nil && c.reader != nil {
		c.reader.Close()
	}
}
