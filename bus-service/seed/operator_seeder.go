package seed

import (
	"encoding/json"
	"os"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

func SeedOperators(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/operators.json")
	if err != nil {
		return err
	}
	var records []model.Operator
	if err := json.Unmarshal(bytes, &records); err != nil {
		return err
	}
	for _, r := range records {
		if err := tx.Where("operator_code = ?", r.OperatorCode).FirstOrCreate(&r).Error; err != nil {
			return err
		}
	}
	return nil
}
