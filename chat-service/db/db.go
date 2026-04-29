package db

import (
	"log"

	"github.com/junaid9001/tripneo/chat-service/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to chat database: %v", err)
	}

	log.Println("Successfully connected to chat database")

	if err := db.AutoMigrate(&models.ChatMessage{}); err != nil {
		log.Fatalf("Failed to migrate chat database: %v", err)
	}

	return db
}
