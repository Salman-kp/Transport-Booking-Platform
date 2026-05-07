package db

import (
	"log"

	"github.com/junaid9001/tripneo/auth-service/config"
	"github.com/junaid9001/tripneo/auth-service/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectPostgres(cfg *config.Config) {

	dsn := cfg.DB_URL

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL:", err)
	}

	if err = db.AutoMigrate(&models.User{}); err != nil {
		log.Fatal("db migration failed")
	}

	SeedSuperAdmins(db, cfg)

	log.Println("Connected to PostgreSQL!")

	DB = db
}
