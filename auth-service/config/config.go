package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APP_PORT string

	REDIS_HOST string
	REDIS_PORT string

	DB_URL string

	JWT_SECRET string
	JWT_EXPIRY string

	RESEND_API_KEY string

	SUPER_ADMINS string
	SUPER_ADMIN_PASSWORD string
}

// load env and initialize config struct
func LoadConfig() *Config {

	_ = godotenv.Load()

	APP_PORT := os.Getenv("APP_PORT")

	REDIS_HOST := os.Getenv("REDIS_HOST")
	REDIS_PORT := os.Getenv("REDIS_PORT")

	DB_URL := os.Getenv("DB_URL")

	JWT_SECRET := os.Getenv("JWT_SECRET")
	JWT_EXPIRY := os.Getenv("JWT_EXPIRY")

	RESEND_API_KEY := os.Getenv("RESEND_API_KEY")

	SUPER_ADMINS := os.Getenv("SUPER_ADMINS")
	SUPER_ADMIN_PASSWORD := os.Getenv("SUPER_ADMIN_PASSWORD")

	cfg := &Config{
		APP_PORT: APP_PORT,

		REDIS_HOST: REDIS_HOST,
		REDIS_PORT: REDIS_PORT,

		DB_URL: DB_URL,

		JWT_SECRET: JWT_SECRET,
		JWT_EXPIRY: JWT_EXPIRY,

		RESEND_API_KEY: RESEND_API_KEY,

		SUPER_ADMINS: SUPER_ADMINS,
		SUPER_ADMIN_PASSWORD: SUPER_ADMIN_PASSWORD,
	}

	return cfg
}
