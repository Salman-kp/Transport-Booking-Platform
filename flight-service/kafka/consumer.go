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

type PaymentRefundedEvent struct {
	BookingID string  `json:"booking_id"`
	PaymentID string  `json:"payment_id"`
	RefundID  string  `json:"refund_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	UserID    string  `json:"user_id"`
	Domain    string  `json:"domain"`
	Status    string  `json:"status"`
	Reason    string  `json:"reason"`
}

type PaymentRefundFailedEvent struct {
	BookingID string  `json:"booking_id"`
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	UserID    string  `json:"user_id"`
	Domain    string  `json:"domain"`
	Status    string  `json:"status"`
	Reason    string  `json:"reason"`
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

func (c *Consumer) ConsumeRefundEvents(ctx context.Context, handler func(PaymentRefundedEvent)) {
	if c == nil || c.reader == nil {
		return
	}
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("[kafka] Error reading refund message: %v", err)
			break
		}

		var evt PaymentRefundedEvent
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			log.Printf("[kafka] Error unmarshaling refund event: %v", err)
			continue
		}

		log.Printf("[kafka] Received refund event for booking: %s", evt.BookingID)
		handler(evt)
	}
}

func (c *Consumer) ConsumeRefundFailedEvents(ctx context.Context, handler func(PaymentRefundFailedEvent)) {
	if c == nil || c.reader == nil {
		return
	}
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("[kafka] Error reading refund-failed message: %v", err)
			break
		}

		var evt PaymentRefundFailedEvent
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			log.Printf("[kafka] Error unmarshaling refund-failed event: %v", err)
			continue
		}

		log.Printf("[kafka] Received refund-failed event for booking: %s", evt.BookingID)
		handler(evt)
	}
}
