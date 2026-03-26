package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APP_PORT                 string
	DB_URL                   string
	REDIS_HOST               string
	REDIS_PORT               string
	PAYMENT_SERVICE_GRPC_URL string
	KAFKA_BROKERS            string
	PROVIDER_API_URL         string
	PROVIDER_API_KEY         string
	JWT_SECRET               string
}

func LoadEnv() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ .env file not found, using system env")
	} else {
		fmt.Println("✅ .env loaded successfully")
	}
}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("❌ Missing required env: %s", key)
	}
	return val
}

func LoadConfig() *Config {
	LoadEnv()

	return &Config{
		APP_PORT:                 mustGetEnv("APP_PORT"),
		DB_URL:                   mustGetEnv("DB_URL"),
		REDIS_HOST:               mustGetEnv("REDIS_HOST"),
		REDIS_PORT:               mustGetEnv("REDIS_PORT"),
		PAYMENT_SERVICE_GRPC_URL: mustGetEnv("PAYMENT_SERVICE_GRPC_URL"),
		KAFKA_BROKERS:            mustGetEnv("KAFKA_BROKERS"),
		PROVIDER_API_URL:         mustGetEnv("PROVIDER_API_URL"),
		PROVIDER_API_KEY:         mustGetEnv("PROVIDER_API_KEY"),
		JWT_SECRET:               mustGetEnv("JWT_SECRET"),
	}
}
