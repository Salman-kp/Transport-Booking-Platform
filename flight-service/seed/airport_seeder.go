package seed

import (
	"encoding/json"
	"os"

	"github.com/junaid9001/tripneo/flight-service/models"
	"gorm.io/gorm"
)

func SeedAirport(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/airports.json")
	if err != nil {
		return err
	}
	var rawAirports []struct {
		Name      string  `json:"name"`
		IataCode  string  `json:"iata_code"`
		City      string  `json:"city"`
		Country   string  `json:"country"`
		TimeZone  string  `json:"time_zone"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := json.Unmarshal(bytes, &rawAirports); err != nil {
		return err
	}
	for _, r := range rawAirports {
		airport := models.Airport{
			Name:      r.Name,
			IataCode:  r.IataCode,
			City:      r.City,
			Country:   r.Country,
			TimeZone:  r.TimeZone,
			Latitude:  r.Latitude,
			Longitude: r.Longitude,
		}
		if err := tx.Where("iata_code = ?", r.IataCode).FirstOrCreate(&airport).Error; err != nil {
			return err
		}
	}
	return nil
}
