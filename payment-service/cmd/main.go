package main

import (
	"log"
	"net"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"github.com/junaid9001/tripneo/payment-service/handlers"
	"github.com/junaid9001/tripneo/payment-service/kafka"
	"github.com/junaid9001/tripneo/payment-service/proto"
	"github.com/junaid9001/tripneo/payment-service/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	_ = godotenv.Load()

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8085"
	}

	stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeSecretKey == "" {
		log.Println("[WARNING] STRIPE_SECRET_KEY is not set. Payment creation will fail.")
	}

	stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if stripeWebhookSecret == "" {
		log.Println("[WARNING] STRIPE_WEBHOOK_SECRET is not set. Webhook verification will fail.")
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		log.Println("[WARNING] KAFKA_BROKERS is not set. Kafka producer will be disabled.")
	}

	producer := kafka.NewProducer(kafkaBrokers)
	defer producer.Close()

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	paymentService := service.NewPaymentService(stripeSecretKey, producer)
	proto.RegisterPaymentServiceServer(grpcServer, paymentService)
	reflection.Register(grpcServer)

	log.Printf("Starting gRPC server on port %s", grpcPort)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	app := fiber.New()
	webhookHandler := handlers.NewWebhookHandler(producer, stripeWebhookSecret)

	app.Post("/webhooks/stripe", webhookHandler.HandleStripeWebhook)

	log.Printf("Starting Stripe Webhook HTTP server on port %s", httpPort)
	if err := app.Listen(":" + httpPort); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
