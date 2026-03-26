package seed

import (
	"encoding/json"
	"os"

	"github.com/junaid9001/tripneo/flight-service/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func SeedAircraftType(tx *gorm.DB) error {
	fileBytes, err := os.ReadFile("data/aircraft_types.json")
	if err != nil {
		return err
	}
	var rawAircrafts []struct {
		Model        string                 `json:"model"`
		Manufacturer string                 `json:"manufacturer"`
		SeatLayout   map[string]interface{} `json:"seat_layout"`
	}
	if err := json.Unmarshal(fileBytes, &rawAircrafts); err != nil {
		return err
	}
	for _, r := range rawAircrafts {
		// Marshal the map into JSON bytes so it matches datatypes.JSON ([]byte)
		layoutBytes, err := json.Marshal(r.SeatLayout)
		if err != nil {
			return err
		}
		aircraft := models.AircraftType{
			Model:        r.Model,
			Manufacturer: r.Manufacturer,
			SeatLayout:   datatypes.JSON(layoutBytes),
		}
		if err := tx.Where("model = ?", r.Model).FirstOrCreate(&aircraft).Error; err != nil {
			return err
		}
	}
	return nil
}
