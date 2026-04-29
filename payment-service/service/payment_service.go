package service

import (
	"context"
	"errors"
	"log"
	"math"
	"strings"

	"github.com/junaid9001/tripneo/payment-service/kafka"
	"github.com/junaid9001/tripneo/payment-service/proto"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/paymentintent"
	"github.com/stripe/stripe-go/v81/refund"
)

type PaymentService struct {
	proto.UnimplementedPaymentServiceServer
	producer *kafka.Producer
}

func NewPaymentService(apiKey string, producer *kafka.Producer) *PaymentService {
	stripe.Key = apiKey
	return &PaymentService{
		producer: producer,
	}
}

func (s *PaymentService) CreateOrder(ctx context.Context, req *proto.CreateOrderRequest) (*proto.CreateOrderResponse, error) {
	log.Printf("Creating Stripe PaymentIntent for Booking: %s, Amount: %f", req.BookingId, req.Amount)

	// stripe amount is in cents
	amountInCents := int64(req.Amount * 100)

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountInCents),
		Currency: stripe.String(req.Currency),
		Metadata: map[string]string{
			"booking_id": req.BookingId,
			"domain":     req.Domain,
			"user_id":    req.UserId,
		},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		log.Printf("[STRIPE ERROR] Failed to create PaymentIntent for Booking %s: %v", req.BookingId, err)
		return nil, errors.New("failed to create stripe payment intent")
	}

	log.Printf("[STRIPE SUCCESS] PaymentIntent %s created for Booking %s", pi.ID, req.BookingId)

	return &proto.CreateOrderResponse{
		StripeClientSecret: pi.ClientSecret,
	}, nil
}

func (s *PaymentService) CreateRefund(ctx context.Context, req *proto.CreateRefundRequest) (*proto.CreateRefundResponse, error) {
	if req.PaymentId == "" {
		return nil, errors.New("payment_id is required for refund")
	}
	if req.Amount <= 0 {
		return nil, errors.New("refund amount must be greater than 0")
	}

	amountInSmallest := int64(math.Round(req.Amount * 100))
	refundReason := normalizeRefundReason(req.Reason)

	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(req.PaymentId),
		Amount:        stripe.Int64(amountInSmallest),
		Reason:        stripe.String(refundReason),
		Metadata: map[string]string{
			"booking_id": req.BookingId,
			"domain":     req.Domain,
			"user_id":    req.UserId,
		},
	}

	rf, err := refund.New(params)
	if err != nil {
		log.Printf("[STRIPE ERROR] Refund creation failed for booking %s, payment %s: %v", req.BookingId, req.PaymentId, err)
		if s.producer != nil {
			s.producer.PublishPaymentRefundFailed(ctx, kafka.PaymentRefundFailedEvent{
				BookingID: req.BookingId,
				PaymentID: req.PaymentId,
				Amount:    req.Amount,
				Currency:  req.Currency,
				UserID:    req.UserId,
				Domain:    req.Domain,
				Status:    "FAILED",
				Reason:    err.Error(),
			})
		}
		return nil, errors.New("failed to create stripe refund")
	}

	log.Printf("[STRIPE SUCCESS] Refund %s created for booking %s (payment %s)", rf.ID, req.BookingId, req.PaymentId)
	if s.producer != nil {
		s.producer.PublishPaymentRefunded(ctx, kafka.PaymentRefundedEvent{
			BookingID: req.BookingId,
			PaymentID: req.PaymentId,
			RefundID:  rf.ID,
			Amount:    req.Amount,
			Currency:  req.Currency,
			UserID:    req.UserId,
			Domain:    req.Domain,
			Status:    string(rf.Status),
			Reason:    refundReason,
		})
	}

	return &proto.CreateRefundResponse{
		RefundId: rf.ID,
		Status:   string(rf.Status),
	}, nil
}

func normalizeRefundReason(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "duplicate":
		return "duplicate"
	case "fraudulent":
		return "fraudulent"
	default:
		return "requested_by_customer"
	}
}
