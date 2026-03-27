package seed

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

// Struct parsing utility translating BusType JSON configuration into hundreds of raw relational postgres mapped seats
func ComputationallyMapSeats(tx *gorm.DB, busInstanceID uuid.UUID, layout []byte) error {
	var config map[string]map[string]int
	if err := json.Unmarshal(layout, &config); err != nil {
		return err
	}

	var seats []model.Seat
	var aSleeper, aSemi, aSeater int

	if sleeper, ok := config["sleeper"]; ok {
		// Populate Lower Berths
		for i := 1; i <= sleeper["lower_berths"]; i++ {
			seats = append(seats, model.Seat{SeatNumber: fmt.Sprintf("L%d", i), SeatType: "sleeper", BerthType: "LOWER", Position: "WINDOW"})
			aSleeper++
		}
		// Populate Upper Berths
		for i := 1; i <= sleeper["upper_berths"]; i++ {
			seats = append(seats, model.Seat{SeatNumber: fmt.Sprintf("U%d", i), SeatType: "sleeper", BerthType: "UPPER", Position: "WINDOW"})
			aSleeper++
		}
	}
	
	if semi, ok := config["semi_sleeper"]; ok {
		// Populate Semi-Sleeper Recliners assuming 4 per row
		for r := 1; r <= semi["rows"]; r++ {
			seats = append(seats, model.Seat{SeatNumber: fmt.Sprintf("SS%dA", r), SeatType: "semi_sleeper", Position: "WINDOW"})
			seats = append(seats, model.Seat{SeatNumber: fmt.Sprintf("SS%dB", r), SeatType: "semi_sleeper", Position: "AISLE"})
			seats = append(seats, model.Seat{SeatNumber: fmt.Sprintf("SS%dC", r), SeatType: "semi_sleeper", Position: "AISLE"})
			seats = append(seats, model.Seat{SeatNumber: fmt.Sprintf("SS%dD", r), SeatType: "semi_sleeper", Position: "WINDOW"})
			aSemi += 4
		}
	}

	if seater, ok := config["seater"]; ok {
		// Populate Seaters assuming 4 per row conventional
		for r := 1; r <= seater["rows"]; r++ {
			seats = append(seats, model.Seat{SeatNumber: fmt.Sprintf("S%dA", r), SeatType: "seater", Position: "WINDOW"})
			seats = append(seats, model.Seat{SeatNumber: fmt.Sprintf("S%dB", r), SeatType: "seater", Position: "AISLE"})
			seats = append(seats, model.Seat{SeatNumber: fmt.Sprintf("S%dC", r), SeatType: "seater", Position: "AISLE"})
			seats = append(seats, model.Seat{SeatNumber: fmt.Sprintf("S%dD", r), SeatType: "seater", Position: "WINDOW"})
			aSeater += 4
		}
	}

	// Batch bind relation mappings and mass-save!
	for i := range seats {
		seats[i].BusInstanceID = busInstanceID
	}

	if len(seats) > 0 {
		if err := tx.Create(&seats).Error; err != nil {
			return err
		}
	}

	// Gracefully update BusInstance capacities immediately
	return tx.Model(&model.BusInstance{}).Where("id = ?", busInstanceID).Updates(map[string]interface{}{
		"available_seater": aSeater,
		"available_semi_sleeper": aSemi,
		"available_sleeper": aSleeper,
	}).Error
}
