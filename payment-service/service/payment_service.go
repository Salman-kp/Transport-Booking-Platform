package service

import (
	"context"
	"errors"
	"log"

	"github.com/junaid9001/tripneo/payment-service/proto"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/paymentintent"
)

type PaymentService struct {
	proto.UnimplementedPaymentServiceServer
}

func NewPaymentService(apiKey string) *PaymentService {
	stripe.Key = apiKey
	return &PaymentService{}
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
