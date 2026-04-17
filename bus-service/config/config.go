package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PORT                            string
	GRPC_PORT                       string
	ENV                             string
	DB_URL                          string
	RUN_SEED_ON_BOOT                string
	REDIS_HOST                      string
	REDIS_PORT                      string
	REDPANDA_BROKERS                string
	REDPANDA_GROUP_ID               string
	PAYMENT_SERVICE_ADDR            string
	QR_SERVICE_ADDR                 string
	HMAC_SECRET                     string
	PNR_SALT                        string
	BOOKING_EXPIRY_MINUTES          string
	SEAT_LOCK_MINUTES               string
	PRICING_ENGINE_INTERVAL_MINUTES string
	GPS_LOCATION_TTL_SECONDS        string
}

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ .env file not found, using system env")
	} else {
		fmt.Println("✅ .env loaded successfully")
	}
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func LoadConfig() *Config {
	LoadEnv()

	return &Config{
		PORT:                            getEnv("PORT", "8083"),
		GRPC_PORT:                       getEnv("GRPC_PORT", "9092"),
		ENV:                             getEnv("ENV", "development"),
		DB_URL:                          getEnv("DB_URL", "host=localhost port=5432 user=postgres dbname=bus_service sslmode=disable"),
		RUN_SEED_ON_BOOT:                getEnv("RUN_SEED_ON_BOOT", "false"),
		REDIS_HOST:                      getEnv("REDIS_HOST", "localhost"),
		REDIS_PORT:                      getEnv("REDIS_PORT", "6379"),
		REDPANDA_BROKERS:                getEnv("REDPANDA_BROKERS", "localhost:19092"),
		REDPANDA_GROUP_ID:               getEnv("REDPANDA_GROUP_ID", "bus-service"),
		PAYMENT_SERVICE_ADDR:            getEnv("PAYMENT_SERVICE_ADDR", "localhost:8085"),
		QR_SERVICE_ADDR:                 getEnv("QR_SERVICE_ADDR", "localhost:8086"),
		HMAC_SECRET:                     getEnv("HMAC_SECRET", "supersecret123"),
		PNR_SALT:                        getEnv("PNR_SALT", "salt123"),
		BOOKING_EXPIRY_MINUTES:          getEnv("BOOKING_EXPIRY_MINUTES", "15"),
		SEAT_LOCK_MINUTES:               getEnv("SEAT_LOCK_MINUTES", "10"),
		PRICING_ENGINE_INTERVAL_MINUTES: getEnv("PRICING_ENGINE_INTERVAL_MINUTES", "15"),
		GPS_LOCATION_TTL_SECONDS:        getEnv("GPS_LOCATION_TTL_SECONDS", "90"),
	}
}
