package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:        getEnv("PORT", "8086"),
		DatabaseURL: getEnv("DATABASE_URL", "host=localhost user=tripneo password=tripneo dbname=chat_db port=5432 sslmode=disable TimeZone=UTC"),
		JWTSecret:   getEnv("JWT_SECRET", "super-secret-key-change-in-prod"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Printf("Warning: Environment variable %s not set, using default.", key)
	return fallback
}
