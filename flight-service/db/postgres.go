package db

import (
	"log"

	"github.com/junaid9001/tripneo/flight-service/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectPostgres(cfg *config.Config) {
	dsn := cfg.DB_URL

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	if err = db.AutoMigrate(); err != nil {
		log.Fatal("db migration failed")
	}

	log.Println("Connected to PostgreSQL!")

	DB = db
}
