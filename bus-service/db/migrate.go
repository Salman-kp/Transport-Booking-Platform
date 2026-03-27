package db

import (
	"github.com/Salman-kp/tripneo/bus-service/config"
	"github.com/Salman-kp/tripneo/bus-service/model"
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
		&model.Operator{},
		&model.BusStop{},
		&model.BusType{},
		&model.Bus{},
		&model.BusInstance{},
		&model.BoardingPoint{},
		&model.DroppingPoint{},
		&model.FareType{},
		&model.Seat{},
		&model.OperatorUser{},
		&model.OperatorInventory{},
		&model.Booking{},
		&model.Passenger{},
		&model.CancellationPolicy{},
		&model.Cancellation{},
		&model.ETicket{},
		&model.PricingRule{},
	); err != nil {
		log.Fatal("DB migration failed:", err)
	}
	log.Println("Connected to PostgreSQL!")
	DB = db
}
