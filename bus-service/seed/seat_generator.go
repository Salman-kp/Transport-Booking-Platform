package seed

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ComputationallyMapSeats translates BusType JSON configuration into raw relational postgres mapped seats.
// It supports both new dynamic layout (rows, left_columns, right_columns) and backward compatibility.
func ComputationallyMapSeats(tx *gorm.DB, busInstanceID uuid.UUID, layout []byte) error {
	type layoutDetail struct {
		Rows         int `json:"rows"`
		LeftColumns  int `json:"left_columns"`
		RightColumns int `json:"right_columns"`
		// Backward compatibility fields
		LowerBerths int `json:"lower_berths"`
		UpperBerths int `json:"upper_berths"`
	}

	var config map[string]layoutDetail
	if err := json.Unmarshal(layout, &config); err != nil {
		log.Printf("[seed ERROR] failed to unmarshal layout for instance %s: %v\n", busInstanceID, err)
		return err
	}

	// 1. Validation: Ensure only one seat type is defined
	activeType := ""
	var activeDetail layoutDetail
	typeCount := 0

	for k, v := range config {
		if v.Rows > 0 || v.LowerBerths > 0 {
			activeType = k
			activeDetail = v
			typeCount++
		}
	}

	if typeCount == 0 {
		log.Printf("[seed] no active layout found for instance %s\n", busInstanceID)
		return nil
	}
	if typeCount > 1 {
		log.Printf("[seed ERROR] multiple seat types detected for instance %s. Aborting.\n", busInstanceID)
		return fmt.Errorf("mixed seat types not supported")
	}

	// 2. Classification Logic
	seatTypeBehavior := activeType
	left, right := activeDetail.LeftColumns, activeDetail.RightColumns

	if left == 1 && right == 1 {
		seatTypeBehavior = "sleeper"
	} else if left == 2 && right == 3 {
		seatTypeBehavior = "seater"
	} else if left == 2 && right == 2 {
		if activeType == "semi_sleeper" {
			seatTypeBehavior = "semi_sleeper"
		} else {
			seatTypeBehavior = "seater"
		}
	}

	// 3. Position Assignment Rule
	getPosition := func(c, l, r int) string {
		total := l + r
		// Seater logic (Right to Left: W, M, A | A, W)
		if total == 5 {
			seq := []string{"WINDOW", "MIDDLE", "AISLE", "AISLE", "WINDOW"}
			return seq[c-1]
		}
		// Semi-Sleeper logic (Right to Left: W, A | A, W)
		if total == 4 {
			seq := []string{"WINDOW", "AISLE", "AISLE", "WINDOW"}
			return seq[c-1]
		}
		// Sleeper logic (Right to Left: W | W)
		if total == 2 {
			return "WINDOW"
		}
		// Fallback for other layouts
		if c == 1 || c == total {
			return "WINDOW"
		}
		if c == l || c == l+1 {
			return "AISLE"
		}
		return "MIDDLE"
	}

	var seats []model.Seat
	var aSleeper, aSemi, aSeater int
	seatNumbers := make(map[string]bool)

	// 4. Generation Logic
	prefix := "S"
	if seatTypeBehavior == "semi_sleeper" {
		prefix = "SS"
	} else if seatTypeBehavior == "sleeper" {
		prefix = "" // Sleeper uses L/U prefixes
	}

	seatCounter := 0

	// Handle New Format (Rows + Columns)
	if activeDetail.Rows > 0 && (left > 0 || right > 0) {
		totalCols := left + right
		for r := 1; r <= activeDetail.Rows; r++ {
			for c := 1; c <= totalCols; c++ {
				pos := getPosition(c, left, right)
				colChar := string('A' + c - 1)

				if seatTypeBehavior == "sleeper" {
					// Lower Berth
					numL := fmt.Sprintf("L%d%s", r, colChar)
					seats = append(seats, model.Seat{
						BusInstanceID: busInstanceID,
						SeatNumber:    numL,
						SeatType:      "sleeper",
						BerthType:     "LOWER",
						Position:      pos,
						Category:      "GENERAL",
					})
					aSleeper++
					seatNumbers[numL] = true

					// Upper Berth
					numU := fmt.Sprintf("U%d%s", r, colChar)
					seats = append(seats, model.Seat{
						BusInstanceID: busInstanceID,
						SeatNumber:    numU,
						SeatType:      "sleeper",
						BerthType:     "UPPER",
						Position:      pos,
						Category:      "GENERAL",
					})
					aSleeper++
					seatNumbers[numU] = true
				} else {
					category := "GENERAL"
					if seatCounter < 8 {
						category = "WOMEN"
					} else if seatCounter < 16 {
						category = "MEN"
					}

					num := fmt.Sprintf("%s%d%s", prefix, r, colChar)
					seats = append(seats, model.Seat{
						BusInstanceID: busInstanceID,
						SeatNumber:    num,
						SeatType:      seatTypeBehavior,
						BerthType:     "LOWER",
						Position:      pos,
						Category:      category,
					})
					seatNumbers[num] = true
					seatCounter++
					if seatTypeBehavior == "seater" {
						aSeater++
					} else {
						aSemi++
					}
				}
			}
		}
	} else {
		// 5. Backward Compatibility (Fixed berth/row counts)
		if activeType == "sleeper" {
			for i := 1; i <= activeDetail.LowerBerths; i++ {
				num := fmt.Sprintf("L%d", i)
				seats = append(seats, model.Seat{BusInstanceID: busInstanceID, SeatNumber: num, SeatType: "sleeper", BerthType: "LOWER", Position: "WINDOW", Category: "GENERAL"})
				aSleeper++
			}
			for i := 1; i <= activeDetail.UpperBerths; i++ {
				num := fmt.Sprintf("U%d", i)
				seats = append(seats, model.Seat{BusInstanceID: busInstanceID, SeatNumber: num, SeatType: "sleeper", BerthType: "UPPER", Position: "WINDOW", Category: "GENERAL"})
				aSleeper++
			}
		} else {
			for i := 1; i <= activeDetail.Rows; i++ {
				category := "GENERAL"
				if i <= 8 {
					category = "WOMEN"
				} else if i <= 16 {
					category = "MEN"
				}
				num := fmt.Sprintf("%s%d", prefix, i)
				seats = append(seats, model.Seat{BusInstanceID: busInstanceID, SeatNumber: num, SeatType: activeType, BerthType: "LOWER", Position: "WINDOW", Category: category})
				if activeType == "seater" {
					aSeater++
				} else {
					aSemi++
				}
			}
		}
	}

	// 6. Final Validation
	if len(seatNumbers) != len(seats) {
		log.Printf("[seed ERROR] duplicate seat numbers detected for instance %s\n", busInstanceID)
		return fmt.Errorf("duplicate seat numbers")
	}

	// 7. Bulk Create Seats
	if len(seats) > 0 {
		if err := tx.Create(&seats).Error; err != nil {
			log.Printf("[seed ERROR] failed to bulk create seats for instance %s: %v\n", busInstanceID, err)
			return err
		}
	}

	// 8. Availability Synchronization
	return tx.Model(&model.BusInstance{}).Where("id = ?", busInstanceID).Updates(map[string]interface{}{
		"available_seater":       aSeater,
		"available_semi_sleeper": aSemi,
		"available_sleeper":      aSleeper,
	}).Error
}
