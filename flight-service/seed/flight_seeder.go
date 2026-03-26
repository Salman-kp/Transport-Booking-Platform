package seed

import (
	"encoding/json"
	"os"
	"time"

	"github.com/junaid9001/tripneo/flight-service/models"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

func SeedFlight(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/flights.json")
	if err != nil {
		return err
	}
	var rawFlights []struct {
		FlightNumber    string  `json:"flight_number"`
		AirlineIata     string  `json:"airline_iata"`
		AircraftModel   string  `json:"aircraft_model"`
		OriginIata      string  `json:"origin_iata"`
		DestinationIata string  `json:"destination_iata"`
		DepartureTime   string  `json:"departure_time"`
		ArrivalTime     string  `json:"arrival_time"`
		DurationMinutes int     `json:"duration_minutes"`
		DaysOfWeek      []int64 `json:"days_of_week"`
		IsActive        bool    `json:"is_active"`
	}
	if err := json.Unmarshal(bytes, &rawFlights); err != nil {
		return err
	}
	for _, r := range rawFlights {
		var airline models.Airline
		if err := tx.Where("iata_code = ?", r.AirlineIata).First(&airline).Error; err != nil {
			return err
		}
		var aircraft models.AircraftType
		if err := tx.Where("model = ?", r.AircraftModel).First(&aircraft).Error; err != nil {
			return err
		}
		var origin models.Airport
		if err := tx.Where("iata_code = ?", r.OriginIata).First(&origin).Error; err != nil {
			return err
		}
		var dest models.Airport
		if err := tx.Where("iata_code = ?", r.DestinationIata).First(&dest).Error; err != nil {
			return err
		}
		depTime, _ := time.Parse("15:04", r.DepartureTime)
		arrTime, _ := time.Parse("15:04", r.ArrivalTime)

		flight := models.Flight{
			FlightNumber:         r.FlightNumber,
			AirlineID:            airline.ID,
			AircraftTypeID:       aircraft.ID,
			OriginAirportID:      origin.ID,
			DestinationAirportID: dest.ID,
			DepartureTime:        depTime,
			ArrivalTime:          arrTime,
			DurationMinutes:      r.DurationMinutes,
			DaysOfWeek:           pq.Int64Array(r.DaysOfWeek),
			IsActive:             r.IsActive,
		}
		if err := tx.Where("flight_number = ?", flight.FlightNumber).FirstOrCreate(&flight).Error; err != nil {
			return err
		}
	}
	return nil
}
