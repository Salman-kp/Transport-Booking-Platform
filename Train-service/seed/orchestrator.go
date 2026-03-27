package seed

import (
	"log"

	"gorm.io/gorm"
)

func SeedAll(db *gorm.DB) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := SeedStations(tx); err != nil {
			log.Println("Error seeding stations:", err)
			return err
		}
		if err := SeedTrains(tx); err != nil {
			log.Println("Error seeding trains:", err)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	log.Println("Train Seeding completed successfully")
	return nil
}
