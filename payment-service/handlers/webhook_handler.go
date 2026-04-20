package handlers

import (
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/payment-service/kafka"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

type WebhookHandler struct {
	kafkaProducer *kafka.Producer
	webhookSecret string
}

func NewWebhookHandler(producer *kafka.Producer, secret string) *WebhookHandler {
	return &WebhookHandler{
		kafkaProducer: producer,
		webhookSecret: secret,
	}
}

func (h *WebhookHandler) HandleStripeWebhook(c fiber.Ctx) error {
	payload := c.Body()
	sigHeader := c.Get("Stripe-Signature")

	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, h.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		log.Printf("Error verifying Stripe webhook: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid signature"})
	}

	if event.Type == "payment_intent.succeeded" {
		var pi stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &pi)
		if err != nil {
			log.Printf("Error parsing payment_intent: %v", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
		}

		bookingID := pi.Metadata["booking_id"]
		domain := pi.Metadata["domain"]
		userID := pi.Metadata["user_id"]
		amount := float64(pi.Amount) / 100.0 // cents to inr

		log.Printf("Payment Succeeded for Booking: %s, Domain: %s", bookingID, domain)

		kafkaEvent := kafka.PaymentCompletedEvent{
			BookingID: bookingID,
			PaymentID: pi.ID,
			Amount:    amount,
			Currency:  string(pi.Currency),
			UserID:    userID,
			Status:    "SUCCESS",
		}

		if domain == "flight" {
			h.kafkaProducer.PublishFlightPaymentCompleted(c.Context(), kafkaEvent)
		}
		// add other domains here as needed
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "processed"})
}
