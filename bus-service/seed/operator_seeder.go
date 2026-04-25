package seed

import (
	"encoding/json"
	"log"
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
		if r.OperatorCode == "" || r.Name == "" {
			log.Printf("[seed] skipping invalid operator: %+v\n", r)
			continue
		}
		if err := tx.Where("operator_code = ?", r.OperatorCode).FirstOrCreate(&r).Error; err != nil {
			log.Printf("[seed] error seeding operator %s: %v\n", r.OperatorCode, err)
			return err
		}
	}
	log.Println("✅ Operator seeding completed")
	return nil
}

