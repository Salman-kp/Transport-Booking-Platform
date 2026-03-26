package seed

import (
	"encoding/json"
	"os"

	"github.com/junaid9001/tripneo/flight-service/models"
	"gorm.io/gorm"
)

func SeedAirline(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/airlines.json")
	if err != nil {
		return err
	}
	var rawAirlines []struct {
		Name     string `json:"name"`
		IataCode string `json:"iata_code"`
		LogoUrl  string `json:"logo_url"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.Unmarshal(bytes, &rawAirlines); err != nil {
		return err
	}
	for _, r := range rawAirlines {
		airline := models.Airline{
			Name:     r.Name,
			IataCode: r.IataCode,
			LogoUrl:  r.LogoUrl,
			IsActive: r.IsActive,
		}
		if err := tx.Where("iata_code = ?", r.IataCode).FirstOrCreate(&airline).Error; err != nil {
			return err
		}
	}
	return nil
}
