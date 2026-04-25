package seed

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

func SeedBusStops(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/bus_stops.json")
	if err != nil {
		return err
	}
	var records []model.BusStop
	if err := json.Unmarshal(bytes, &records); err != nil {
		return err
	}
	for _, r := range records {
		if r.Name == "" || r.City == "" {
			log.Printf("[seed] skipping invalid bus stop: %+v\n", r)
			continue
		}
		if err := tx.Where("name = ? AND city = ?", r.Name, r.City).FirstOrCreate(&r).Error; err != nil {
			log.Printf("[seed] error seeding bus stop %s: %v\n", r.Name, err)
			return err
		}
	}
	log.Println("✅ Bus stop seeding completed")
	return nil
}

func SeedBusTypes(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/bus_types.json")
	if err != nil {
		return err
	}
	var records []model.BusType
	if err := json.Unmarshal(bytes, &records); err != nil {
		return err
	}
	for _, r := range records {
		if r.Name == "" {
			log.Printf("[seed] skipping invalid bus type: %+v\n", r)
			continue
		}
		if err := tx.Where("name = ?", r.Name).FirstOrCreate(&r).Error; err != nil {
			log.Printf("[seed] error seeding bus type %s: %v\n", r.Name, err)
			return err
		}
	}
	log.Println("✅ Bus type seeding completed")
	return nil
}

// normalizeTime ensures time values are in "1970-01-01THH:MM:SSZ" format for PSQL compatibility.
func normalizeTime(t string) string {
	if t == "" {
		t = "00:00:00"
	}
	// If it already contains T and Z, it's likely a full timestamp
	if strings.Contains(t, "T") && strings.Contains(t, "Z") {
		return t
	}

	parts := strings.Split(t, ":")
	hour, min, sec := "00", "00", "00"

	if len(parts) >= 1 {
		hour = parts[0]
		if len(hour) == 1 {
			hour = "0" + hour
		}
	}
	if len(parts) >= 2 {
		min = parts[1]
	}
	if len(parts) >= 3 {
		sec = parts[2]
	}

	// Output as 1970-01-01Txx:xx:xxZ (ISO-8601)
	return fmt.Sprintf("1970-01-01T%s:%s:%sZ", hour, min, sec)
}

func SeedBuses(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/buses.json")
	if err != nil {
		return err
	}
	var raw []struct {
		BusNumber       string `json:"bus_number"`
		OperatorCode    string `json:"operator_code"`
		BusTypeName     string `json:"bus_type"`
		OriginStop      string `json:"origin_stop"`
		DestinationStop string `json:"destination_stop"`
		DepartureTime   string `json:"departure_time"`
		ArrivalTime     string `json:"arrival_time"`
		DurationMinutes int    `json:"duration_minutes"`
		DaysOfWeek      []int  `json:"days_of_week"`
		DistanceKm      int    `json:"distance_km"`
		IsActive        bool   `json:"is_active"`
	}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return err
	}

	insertedCount := 0
	skippedCount := 0
	failedCount := 0

	for _, r := range raw {
		// 0. Basic Validation
		if r.DistanceKm < 0 || r.DurationMinutes < 0 {
			log.Printf("[seed] skipping invalid bus %s: negative distance or duration\n", r.BusNumber)
			skippedCount++
			continue
		}

		// 1. Validate Referenced Entities (Production-grade validation)
		var op model.Operator
		if err := tx.Where("operator_code = ?", r.OperatorCode).First(&op).Error; err != nil {
			log.Printf("[seed] skipping bus %s: operator %s not found in database\n", r.BusNumber, r.OperatorCode)
			skippedCount++
			continue
		}

		var bt model.BusType
		if err := tx.Where("name = ?", r.BusTypeName).First(&bt).Error; err != nil {
			log.Printf("[seed] skipping bus %s: bus type %s not found in database\n", r.BusNumber, r.BusTypeName)
			skippedCount++
			continue
		}

		var oStop model.BusStop
		if err := tx.Where("name = ?", r.OriginStop).First(&oStop).Error; err != nil {
			log.Printf("[seed] skipping bus %s: origin stop %s not found in database\n", r.BusNumber, r.OriginStop)
			skippedCount++
			continue
		}

		var dStop model.BusStop
		if err := tx.Where("name = ?", r.DestinationStop).First(&dStop).Error; err != nil {
			log.Printf("[seed] skipping bus %s: destination stop %s not found in database\n", r.BusNumber, r.DestinationStop)
			skippedCount++
			continue
		}

		days, _ := json.Marshal(r.DaysOfWeek)

		// 2. Prepare Bus Model with Full Timestamp (1970-01-01 reference)
		bus := model.Bus{
			BusNumber:         r.BusNumber,
			OperatorID:        op.ID,
			BusTypeID:         bt.ID,
			OriginStopID:      oStop.ID,
			DestinationStopID: dStop.ID,
			DepartureTime:     normalizeTime(r.DepartureTime),
			ArrivalTime:       normalizeTime(r.ArrivalTime),
			DurationMinutes:   r.DurationMinutes,
			DaysOfWeek:        days,
			DistanceKM:        r.DistanceKm,
			IsActive:          r.IsActive,
		}

		// 3. Idempotent Insert
		if err := tx.Where("bus_number = ?", r.BusNumber).FirstOrCreate(&bus).Error; err != nil {
			log.Printf("[seed ERROR] failed to seed bus %s into DB: %v\n", r.BusNumber, err)
			failedCount++
			// Note: In Postgres, this will likely abort the current transaction.
			continue
		}
		insertedCount++
	}

	log.Printf("✅ Bus seeding summary: %d inserted, %d failed, %d skipped due to missing dependencies\n",
		insertedCount, failedCount, skippedCount)

	if insertedCount == 0 && len(raw) > 0 {
		return fmt.Errorf("seeding failed: all %d bus records were rejected. Check logs for details", len(raw))
	}

	return nil
}



