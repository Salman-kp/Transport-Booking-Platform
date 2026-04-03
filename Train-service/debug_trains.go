package main

import (
	"log"

	"github.com/nabeel-mp/tripneo/train-service/config"
	"github.com/nabeel-mp/tripneo/train-service/db"
	"github.com/nabeel-mp/tripneo/train-service/models"
)

func main() {
	cfg := config.LoadConfig()
	db.ConnectPostgres(cfg)

	var trains []models.Train
	if err := db.DB.Preload("Stops").Find(&trains).Error; err != nil {
		log.Fatalf("Error: %v", err)
	}
	
	for _, t := range trains {
		log.Printf("Train: %s, IsActive: %t, Stops: %d, Days: %v", t.TrainNumber, t.IsActive, len(t.Stops), t.DaysOfWeek)
	}
}
