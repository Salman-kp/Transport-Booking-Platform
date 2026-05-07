package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APP_PORT                 string
	DB_URL                   string
	REDIS_URL                string
	RUN_SEED_ON_BOOT         string
	PAYMENT_SERVICE_GRPC_URL string
	KAFKA_BROKERS            string
	AVIATIONSTACK_API_KEY    string
	QR_PUBLIC_BASE_URL       string
	QR_SIGNING_SECRET        string
	AUTH_SERVICE_URL         string
	RESEND_API_KEY           string
	RESEND_FROM_ADDRESS      string
}

func LoadConfig() *Config {
	_ = godotenv.Load()
	config := &Config{
		APP_PORT:                 os.Getenv("APP_PORT"),
		DB_URL:                   os.Getenv("DB_URL"),
		REDIS_URL:                os.Getenv("REDIS_URL"),
		RUN_SEED_ON_BOOT:         os.Getenv("RUN_SEED_ON_BOOT"),
		PAYMENT_SERVICE_GRPC_URL: os.Getenv("PAYMENT_SERVICE_GRPC_URL"),
		KAFKA_BROKERS:            os.Getenv("KAFKA_BROKERS"),
		AVIATIONSTACK_API_KEY:    os.Getenv("AVIATIONSTACK_API_KEY"),
		QR_PUBLIC_BASE_URL:       os.Getenv("QR_PUBLIC_BASE_URL"),
		QR_SIGNING_SECRET:        os.Getenv("QR_SIGNING_SECRET"),
		AUTH_SERVICE_URL:         os.Getenv("AUTH_SERVICE_URL"),
		RESEND_API_KEY:           os.Getenv("RESEND_API_KEY"),
		RESEND_FROM_ADDRESS:      os.Getenv("RESEND_FROM_ADDRESS"),
	}
	if config.PAYMENT_SERVICE_GRPC_URL == "" {
		config.PAYMENT_SERVICE_GRPC_URL = "localhost:50051"
	}
	if config.KAFKA_BROKERS == "" {
		config.KAFKA_BROKERS = "localhost:9092"
	}
	if config.QR_PUBLIC_BASE_URL == "" {
		config.QR_PUBLIC_BASE_URL = "http://localhost:8080/api/qr/generate"
	}
	if config.QR_SIGNING_SECRET == "" {
		config.QR_SIGNING_SECRET = "dev-insecure-change-me"
	}
	return config
}
