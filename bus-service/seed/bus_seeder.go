package seed

import (
	"encoding/json"
	"log"
	"os"

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
		if err := tx.Where("name = ? AND city = ?", r.Name, r.City).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
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
		if err := tx.Where("name = ?", r.Name).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
	return nil
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
	for _, r := range raw {
		var op model.Operator
		if err := tx.Where("operator_code = ?", r.OperatorCode).First(&op).Error; err != nil {
			log.Println("Operator not found:", r.OperatorCode)
			continue
		}
		var bt model.BusType
		if err := tx.Where("name = ?", r.BusTypeName).First(&bt).Error; err != nil {
			log.Println("BusType not found:", r.BusTypeName)
			continue
		}
		var os model.BusStop
		if err := tx.Where("name = ?", r.OriginStop).First(&os).Error; err != nil {
			log.Println("Origin stop not found:", r.OriginStop)
			continue
		}
		var ds model.BusStop
		if err := tx.Where("name = ?", r.DestinationStop).First(&ds).Error; err != nil {
			log.Println("Destination stop not found:", r.DestinationStop)
			continue
		}
		days, _ := json.Marshal(r.DaysOfWeek)

		bus := model.Bus{
			BusNumber:         r.BusNumber,
			OperatorID:        op.ID,
			BusTypeID:         bt.ID,
			OriginStopID:      os.ID,
			DestinationStopID: ds.ID,
			DepartureTime:     "1970-01-01T" + r.DepartureTime + ":00Z",
			ArrivalTime:       "1970-01-01T" + r.ArrivalTime + ":00Z",
			DurationMinutes:   r.DurationMinutes,
			DaysOfWeek:        days,
			DistanceKM:        r.DistanceKm,
			IsActive:          r.IsActive,
		}
		if err := tx.Where("bus_number = ?", r.BusNumber).FirstOrCreate(&bus).Error; err != nil {
			return err
		}
	}
	return nil
}
