package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

// Producer wraps a kafka-go writer.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a Kafka producer. Returns nil if broker is empty (local dev without Kafka).
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

// Publish sends a message to a Kafka topic.
// Safe to call on a nil Producer (Kafka disabled in local dev).
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

// PublishBookingCreated publishes a train.booking.created event.
func (p *Producer) PublishBookingCreated(ctx context.Context, evt BookingCreatedEvent) {
	p.Publish(ctx, TopicBookingCreated, evt.BookingID, toJSON(evt))
}

// PublishBookingConfirmed publishes a train.booking.confirmed event.
func (p *Producer) PublishBookingConfirmed(ctx context.Context, evt BookingConfirmedEvent) {
	p.Publish(ctx, TopicBookingConfirmed, evt.BookingID, toJSON(evt))
}

// PublishBookingCancelled publishes a train.booking.cancelled event.
func (p *Producer) PublishBookingCancelled(ctx context.Context, evt BookingCancelledEvent) {
	p.Publish(ctx, TopicBookingCancelled, evt.BookingID, toJSON(evt))
}

// PublishBookingExpired publishes a train.booking.expired event.
func (p *Producer) PublishBookingExpired(ctx context.Context, evt BookingExpiredEvent) {
	p.Publish(ctx, TopicBookingExpired, evt.BookingID, toJSON(evt))
}

// PublishNotification publishes a notification.send event.
func (p *Producer) PublishNotification(ctx context.Context, evt NotificationEvent) {
	p.Publish(ctx, TopicNotificationSend, evt.RecipientUserID, toJSON(evt))
}

// Close shuts down the producer gracefully.
func (p *Producer) Close() {
	if p != nil && p.writer != nil {
		p.writer.Close()
	}
}
