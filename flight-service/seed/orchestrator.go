package seed

import (
	"log"

	"gorm.io/gorm"
)

func SeedAll(db *gorm.DB) error {
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := SeedAirport(tx); err != nil {
			log.Println("Error seeding airports:", err)
			return err
		}
		if err := SeedAirline(tx); err != nil {
			log.Println("Error seeding airlines:", err)
			return err
		}
		if err := SeedAircraftType(tx); err != nil {
			log.Println("Error seeding aircraft types:", err)
			return err
		}
		if err := SeedFlight(tx); err != nil {
			log.Println("Error seeding flights:", err)
			return err
		}
		return nil
	})
	if err != nil {
		log.Println("Seeding Data Failed, err=", err)
		return err
	}
	log.Println("Seeding completed successfully")
	return nil
}
