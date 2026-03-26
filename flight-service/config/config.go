package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APP_PORT         string
	DB_URL           string
	RUN_SEED_ON_BOOT string
}

func LoadConfig() *Config {
	_ = godotenv.Load()
	APP_PORT := os.Getenv("APP_PORT")
	DB_URL := os.Getenv("DB_URL")
	RUN_SEED_ON_BOOT := os.Getenv("RUN_SEED_ON_BOOT")

	config := &Config{
		APP_PORT:         APP_PORT,
		DB_URL:           DB_URL,
		RUN_SEED_ON_BOOT: RUN_SEED_ON_BOOT,
	}
	return config
}
