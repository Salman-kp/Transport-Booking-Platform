package redpanda

import (
	"context"
	"log"

	kafkago "github.com/segmentio/kafka-go"
)

// Producer wraps a kafka-go writer.
// All Publish* methods are safe to call on a nil Producer (Redpanda disabled in local dev).
type Producer struct {
	writer *kafkago.Writer
}

// NewProducer creates a Redpanda producer connected to the given broker.
// Returns nil if broker is empty — all publish calls are no-ops.
func NewProducer(broker string) *Producer {
	if broker == "" {
		log.Println("[bus-redpanda] No broker configured — Redpanda producer disabled")
		return nil
	}
	w := &kafkago.Writer{
		Addr:                   kafkago.TCP(broker),
		Balancer:               &kafkago.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
	log.Printf("[bus-redpanda] Producer connected to %s", broker)
	return &Producer{writer: w}
}

// publish is the internal sender. Silently drops if producer is nil.
func (p *Producer) publish(ctx context.Context, topic, key string, value []byte) {
	if p == nil || p.writer == nil {
		return
	}
	msg := kafkago.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("[bus-redpanda] Failed to publish to %s: %v", topic, err)
	} else {
		log.Printf("produced event %s", topic)
	}
}

// ── Produced event publishers ─────────────────────────────────────────────────

func (p *Producer) PublishBookingCreated(ctx context.Context, evt BookingCreatedEvent) {
	p.publish(ctx, TopicBusBookingCreated, evt.BookingID, toJSON(evt))
}

func (p *Producer) PublishBookingConfirmed(ctx context.Context, evt BookingConfirmedEvent) {
	p.publish(ctx, TopicBusBookingConfirmed, evt.BookingID, toJSON(evt))
}

func (p *Producer) PublishBookingCancelled(ctx context.Context, evt BookingCancelledEvent) {
	p.publish(ctx, TopicBusBookingCancelled, evt.BookingID, toJSON(evt))
}

func (p *Producer) PublishBookingExpired(ctx context.Context, evt BookingExpiredEvent) {
	p.publish(ctx, TopicBusBookingExpired, evt.BookingID, toJSON(evt))
}

func (p *Producer) PublishPaymentInitiated(ctx context.Context, evt PaymentInitiatedEvent) {
	p.publish(ctx, TopicBusPaymentInitiated, evt.BookingID, toJSON(evt))
}

func (p *Producer) PublishNotification(ctx context.Context, evt NotificationEvent) {
	p.publish(ctx, TopicNotificationSend, evt.RecipientUserID, toJSON(evt))
}

// ── Payment Simulation Publishers (for verification) ──────────────────────────

func (p *Producer) PublishPaymentCompleted(ctx context.Context, evt PaymentCompletedEvent) {
	p.publish(ctx, TopicPaymentCompleted, evt.BookingID, toJSON(evt))
}

func (p *Producer) PublishPaymentFailed(ctx context.Context, evt PaymentFailedEvent) {
	p.publish(ctx, TopicPaymentFailed, evt.BookingID, toJSON(evt))
}

func (p *Producer) PublishPaymentRefunded(ctx context.Context, evt PaymentRefundedEvent) {
	p.publish(ctx, TopicPaymentRefunded, evt.BookingID, toJSON(evt))
}

// Close gracefully shuts down the writer.
func (p *Producer) Close() {
	if p != nil && p.writer != nil {
		p.writer.Close()
	}
}
