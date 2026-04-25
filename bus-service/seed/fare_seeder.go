package seed

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

func SeedFareTypes(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/fare_types.json")
	if err != nil {
		return err
	}
	var raw []struct {
		BusNumber       string  `json:"bus_number"`
		TravelDate      string  `json:"travel_date"`
		Name            string  `json:"name"`
		SeatType        string  `json:"seat_type"`
		Price           float64 `json:"price"`
		IsRefundable    bool    `json:"is_refundable"`
		CancellationFee float64 `json:"cancellation_fee"`
		DateChangeFee   float64 `json:"date_change_fee"`
		SeatsAvailable  int     `json:"seats_available"`
	}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return err
	}

	for _, r := range raw {
		// 0. Validation
		if r.Price < 0 || r.CancellationFee < 0 || r.DateChangeFee < 0 || r.SeatsAvailable < 0 {
			log.Printf("[seed] skipping invalid fare: negative values found for %s\n", r.BusNumber)
			continue
		}

		var bus model.Bus
		if err := tx.Where("bus_number = ?", r.BusNumber).First(&bus).Error; err != nil {
			log.Printf("[seed] skipping fare: bus %s not found\n", r.BusNumber)
			continue
		}

		var inst model.BusInstance
		tDate, _ := time.Parse("2006-01-02", r.TravelDate)
		if err := tx.Where("bus_id = ? AND travel_date = ?", bus.ID, tDate).First(&inst).Error; err != nil {
			log.Printf("[seed] skipping fare: bus instance not found for %s on %s\n", r.BusNumber, r.TravelDate)
			continue
		}

		// 1. Validate Seat Type Availability in BusInstance
		availableCount := 0
		st := strings.ToLower(r.SeatType)
		switch st {
		case "seater":
			availableCount = inst.AvailableSeater
		case "semi_sleeper", "semi-sleeper":
			availableCount = inst.AvailableSemiSleeper
		case "sleeper":
			availableCount = inst.AvailableSleeper
		default:
			log.Printf("[seed] skipping fare: unknown seat type %s for bus %s\n", r.SeatType, r.BusNumber)
			continue
		}

		if availableCount <= 0 {
			log.Printf("[seed] skipping fare: %s does not have %s seats available on %s\n",
				r.BusNumber, r.SeatType, r.TravelDate)
			continue
		}

		// 2. Cap SeatsAvailable to actual availability
		seatsToAssign := r.SeatsAvailable
		if seatsToAssign > availableCount {
			seatsToAssign = availableCount
		}

		ft := model.FareType{
			BusInstanceID:   inst.ID,
			SeatType:        r.SeatType,
			Name:            r.Name,
			Price:           r.Price,
			IsRefundable:    r.IsRefundable,
			CancellationFee: r.CancellationFee,
			DateChangeFee:   r.DateChangeFee,
			SeatsAvailable:  seatsToAssign,
		}

		// 3. Idempotent Insert using unique constraints
		if err := tx.Where("bus_instance_id = ? AND name = ? AND seat_type = ?", inst.ID, r.Name, r.SeatType).FirstOrCreate(&ft).Error; err != nil {
			log.Printf("[seed] error creating fare for %s: %v\n", r.BusNumber, err)
			return err
		}
	}

	log.Println("✅ Fare type seeding completed successfully")
	return nil
}
