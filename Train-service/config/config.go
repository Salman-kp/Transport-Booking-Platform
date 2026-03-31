package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APP_PORT string

	DB_URL string

	REDIS_HOST string
	REDIS_PORT string

	JWT_SECRET string

	GRPC_PORT            string
	PAYMENT_SERVICE_GRPC string

	KAFKA_BROKER string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		APP_PORT: os.Getenv("APP_PORT"),

		DB_URL: os.Getenv("DB_URL"),

		REDIS_HOST: os.Getenv("REDIS_HOST"),
		REDIS_PORT: os.Getenv("REDIS_PORT"),

		JWT_SECRET: os.Getenv("JWT_SECRET"),

		GRPC_PORT:            os.Getenv("GRPC_PORT"),
		PAYMENT_SERVICE_GRPC: os.Getenv("PAYMENT_SERVICE_GRPC"),

		KAFKA_BROKER: os.Getenv("KAFKA_BROKER"),
	}
}
