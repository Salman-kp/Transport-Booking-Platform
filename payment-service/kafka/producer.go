package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(broker string) *Producer {
	if broker == "" {
		log.Println("[kafka] No broker configured — Kafka producer disabled")
		return nil
	}
	w := &kafka.Writer{
		Addr:                   kafka.TCP(broker),
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
	log.Printf("[kafka] Producer connected to %s", broker)
	return &Producer{writer: w}
}

func (p *Producer) Publish(ctx context.Context, topic string, key string, value []byte) {
	if p == nil || p.writer == nil {
		return
	}
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("[kafka] Failed to publish to %s: %v", topic, err)
	}
}

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

func (p *Producer) PublishFlightPaymentCompleted(ctx context.Context, evt PaymentCompletedEvent) {
	data, _ := json.Marshal(evt)
	p.Publish(ctx, "flight-payment-topic", evt.BookingID, data)
}

func (p *Producer) PublishPaymentRefunded(ctx context.Context, evt PaymentRefundedEvent) {
	data, _ := json.Marshal(evt)
	p.Publish(ctx, "payment.refunded", evt.BookingID, data)
}

func (p *Producer) PublishPaymentRefundFailed(ctx context.Context, evt PaymentRefundFailedEvent) {
	data, _ := json.Marshal(evt)
	p.Publish(ctx, "payment.refund_failed", evt.BookingID, data)
func (p *Producer) PublishBusPaymentCompleted(ctx context.Context, evt PaymentCompletedEvent) {
	data, _ := json.Marshal(evt)
	p.Publish(ctx, "bus-payment-topic", evt.BookingID, data)
}

func (p *Producer) Close() {
	if p != nil && p.writer != nil {
		p.writer.Close()
	}
}
