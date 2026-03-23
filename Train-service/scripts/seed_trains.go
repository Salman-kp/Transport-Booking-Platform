//go:build ignore

package main

import (
	"log"
	"os"

	"github.com/lib/pq"
	"github.com/nabeel-mp/tripneo/train-service/config"
	"github.com/nabeel-mp/tripneo/train-service/db"
	"github.com/nabeel-mp/tripneo/train-service/models"
)

func main() {
	// Load .env from train-service root
	if err := os.Chdir(".."); err != nil {
		log.Fatal(err)
	}
	cfg := config.LoadConfig()
	db.ConnectPostgres(cfg)

	log.Println("Seeding train templates...")

	trains := []models.Train{
		{
			TrainNumber:        "12678",
			TrainName:          "Ernakulam - Chennai Express",
			OriginStation:      "ERS",
			DestinationStation: "MAS",
			DepartureTime:      "20:30",
			ArrivalTime:        "05:30",
			DurationMinutes:    540,
			DaysOfWeek:         pq.Int32Array{1, 2, 3, 4, 5, 6, 7},
			IsActive:           true,
		},
		{
			TrainNumber:        "12601",
			TrainName:          "Chennai Mail",
			OriginStation:      "MAS",
			DestinationStation: "CBE",
			DepartureTime:      "21:00",
			ArrivalTime:        "05:00",
			DurationMinutes:    480,
			DaysOfWeek:         pq.Int32Array{1, 2, 3, 4, 5, 6, 7},
			IsActive:           true,
		},
		{
			TrainNumber:        "16301",
			TrainName:          "Venad Express",
			OriginStation:      "TVC",
			DestinationStation: "SRR",
			DepartureTime:      "09:45",
			ArrivalTime:        "18:15",
			DurationMinutes:    510,
			DaysOfWeek:         pq.Int32Array{1, 2, 3, 4, 5, 6, 7},
			IsActive:           true,
		},
		{
			TrainNumber:        "12082",
			TrainName:          "Jan Shatabdi Express",
			OriginStation:      "CBE",
			DestinationStation: "MAS",
			DepartureTime:      "06:00",
			ArrivalTime:        "12:30",
			DurationMinutes:    390,
			DaysOfWeek:         pq.Int32Array{1, 2, 3, 4, 5, 6, 7},
			IsActive:           true,
		},
		{
			TrainNumber:        "16526",
			TrainName:          "Island Express",
			OriginStation:      "KGQ",
			DestinationStation: "TVC",
			DepartureTime:      "14:30",
			ArrivalTime:        "22:45",
			DurationMinutes:    495,
			DaysOfWeek:         pq.Int32Array{1, 2, 4, 5, 6},
			IsActive:           true,
		},
		{
			TrainNumber:        "22637",
			TrainName:          "West Coast Express",
			OriginStation:      "MAS",
			DestinationStation: "ERS",
			DepartureTime:      "19:15",
			ArrivalTime:        "08:00",
			DurationMinutes:    765,
			DaysOfWeek:         pq.Int32Array{2, 4, 6},
			IsActive:           true,
		},
		{
			TrainNumber:        "12625",
			TrainName:          "Kerala Express",
			OriginStation:      "NDLS",
			DestinationStation: "TVC",
			DepartureTime:      "11:35",
			ArrivalTime:        "10:30",
			DurationMinutes:    1375,
			DaysOfWeek:         pq.Int32Array{1, 3, 5, 7},
			IsActive:           true,
		},
		{
			TrainNumber:        "16381",
			TrainName:          "Kanyakumari Express",
			OriginStation:      "CST",
			DestinationStation: "CAPE",
			DepartureTime:      "01:00",
			ArrivalTime:        "19:45",
			DurationMinutes:    1125,
			DaysOfWeek:         pq.Int32Array{3, 6},
			IsActive:           true,
		},
	}

	var created, skipped int
	for _, t := range trains {
		var existing models.Train
		result := db.DB.Where("train_number = ?", t.TrainNumber).First(&existing)
		if result.Error == nil {
			log.Printf("  SKIP  %s — %s (already exists)", t.TrainNumber, t.TrainName)
			skipped++
			continue
		}
		if err := db.DB.Create(&t).Error; err != nil {
			log.Printf("  ERROR %s — %v", t.TrainNumber, err)
			continue
		}
		log.Printf("  OK    %s — %s | %s→%s | %s | days:%v",
			t.TrainNumber, t.TrainName,
			t.OriginStation, t.DestinationStation,
			t.DepartureTime, []int32(t.DaysOfWeek),
		)
		created++
	}

	log.Printf("\nDone. Created: %d  Skipped: %d", created, skipped)
	log.Println("Next step: run the instance generator to create schedules + inventory.")
}
