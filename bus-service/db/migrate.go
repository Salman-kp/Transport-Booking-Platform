package db

import (
	"github.com/Salman-kp/tripneo/bus-service/config"
	"github.com/Salman-kp/tripneo/bus-service/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

var DB *gorm.DB

func ConnectPostgres(cfg *config.Config) {
	db, err := gorm.Open(postgres.Open(cfg.DB_URL), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}
	if err = db.AutoMigrate(
		&model.Bus{},
		&model.Route{},
		&model.Schedule{},
		&model.Seat{},
		&model.Booking{},
		&model.BookingSeat{},
		&model.BusTracking{},
		&model.Inventory{},
		&model.QRScan{},
	); err != nil {
		log.Fatal("DB migration failed:", err)
	}
	log.Println("Connected to PostgreSQL!")
	DB = db
}
