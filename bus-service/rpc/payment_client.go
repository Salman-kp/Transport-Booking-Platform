package rpc

import (
	"context"
	"log"

	"github.com/Salman-kp/tripneo/bus-service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type PaymentClient struct {
	client proto.PaymentServiceClient
	conn   *grpc.ClientConn
}

func NewPaymentClient(address string) (*PaymentClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := proto.NewPaymentServiceClient(conn)
	return &PaymentClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *PaymentClient) CreateOrder(ctx context.Context, bookingID string, amount float64, currency string, userID string) (string, error) {
	req := &proto.CreateOrderRequest{
		BookingId: bookingID,
		Amount:    amount,
		Currency:  currency,
		Domain:    "bus",
		UserId:    userID,
	}

	resp, err := c.client.CreateOrder(ctx, req)
	if err != nil {
		log.Printf("gRPC: Failed to create order: %v", err)
		return "", err
	}

	return resp.StripeClientSecret, nil
}

func (c *PaymentClient) CreateRefund(ctx context.Context, bookingID, paymentID string, amount float64, currency, userID, reason string) (string, error) {
	req := &proto.CreateRefundRequest{
		BookingId: bookingID,
		PaymentId: paymentID,
		Amount:    amount,
		Currency:  currency,
		Domain:    "bus",
		UserId:    userID,
		Reason:    reason,
	}
	resp, err := c.client.CreateRefund(ctx, req)
	if err != nil {
		log.Printf("gRPC: Failed to create refund: %v", err)
		return "", err
	}
	return resp.RefundId, nil
}

func (c *PaymentClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}
