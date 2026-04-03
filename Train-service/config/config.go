package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	APP_PORT string
	ENV      string

	DB_URL string

	REDIS_HOST string
	REDIS_PORT string

	JWT_SECRET  string
	HMAC_SECRET string

	GRPC_PORT            string
	PAYMENT_SERVICE_GRPC string
	QR_SERVICE_GRPC      string

	KAFKA_BROKER   string
	KAFKA_GROUP_ID string

	RAILWAY_API_KEY      string
	RAILWAY_API_BASE_URL string

	BookingExpiryMinutes       int
	SeatLockMinutes            int
	PricingEngineIntervalMins  int
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		APP_PORT: getEnv("APP_PORT", "8082"),
		ENV:      getEnv("ENV", "development"),

		DB_URL: os.Getenv("DB_URL"),

		REDIS_HOST: getEnv("REDIS_HOST", "localhost"),
		REDIS_PORT: getEnv("REDIS_PORT", "6379"),

		JWT_SECRET:  os.Getenv("JWT_SECRET"),
		HMAC_SECRET: getEnv("HMAC_SECRET", "default-hmac-secret-change-in-production"),

		GRPC_PORT:            getEnv("GRPC_PORT", "50052"),
		PAYMENT_SERVICE_GRPC: getEnv("PAYMENT_SERVICE_GRPC", "localhost:50051"),
		QR_SERVICE_GRPC:      getEnv("QR_SERVICE_GRPC", "localhost:50053"),

		KAFKA_BROKER:   getEnv("KAFKA_BROKER", ""),
		KAFKA_GROUP_ID: getEnv("KAFKA_GROUP_ID", "train-service"),

		RAILWAY_API_KEY:      os.Getenv("RAILWAY_API_KEY"),
		RAILWAY_API_BASE_URL: getEnv("RAILWAY_API_BASE_URL", "https://api.railwayapi.com/v2"),

		BookingExpiryMinutes:      getEnvInt("BOOKING_EXPIRY_MINUTES", 15),
		SeatLockMinutes:           getEnvInt("SEAT_LOCK_MINUTES", 10),
		PricingEngineIntervalMins: getEnvInt("PRICING_ENGINE_INTERVAL_MINUTES", 15),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
