package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APP_PORT string

	REDIS_HOST string
	REDIS_PORT string

	FRONTEND_URL       string
	AUTH_SERVICE_URL   string
	FLIGHT_SERVICE_URL string
	TRAIN_SERVICE_URL  string

	JWT_SECRET string
}

// load env and initialize config struct
func LoadConfig() *Config {

	_ = godotenv.Load()

	APP_PORT := os.Getenv("APP_PORT")

	REDIS_HOST := os.Getenv("REDIS_HOST")
	REDIS_PORT := os.Getenv("REDIS_PORT")

	AUTH_SERVICE_URL := os.Getenv("AUTH_SERVICE_URL")
	FLIGHT_SERVICE_URL := os.Getenv("FLIGHT_SERVICE_URL")
	TRAIN_SERVICE_URL := os.Getenv("TRAIN_SERVICE_URL")
	FRONTEND_URL := os.Getenv("FRONTEND_URL")

	JWT_SECRET := os.Getenv("JWT_SECRET")

	cfg := &Config{
		APP_PORT:           APP_PORT,
		REDIS_HOST:         REDIS_HOST,
		REDIS_PORT:         REDIS_PORT,
		AUTH_SERVICE_URL:   AUTH_SERVICE_URL,
		FLIGHT_SERVICE_URL: FLIGHT_SERVICE_URL,
		TRAIN_SERVICE_URL:  TRAIN_SERVICE_URL,
		JWT_SECRET:         JWT_SECRET,
		FRONTEND_URL:       FRONTEND_URL,
	}

	return cfg
}
