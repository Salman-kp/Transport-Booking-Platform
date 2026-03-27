package seed

import (
	"log"

	"gorm.io/gorm"
)

// SeedAll runs all independent sub-seeders with transaction + logs
func SeedAll(db *gorm.DB) error {

	err := db.Transaction(func(tx *gorm.DB) error {

		if err := SeedOperators(tx); err != nil {
			log.Println("Error seeding operators:", err)
			return err
		}

		if err := SeedBusStops(tx); err != nil {
			log.Println("Error seeding bus stops:", err)
			return err
		}

		if err := SeedBusTypes(tx); err != nil {
			log.Println("Error seeding bus types:", err)
			return err
		}

		if err := SeedBuses(tx); err != nil {
			log.Println("Error seeding buses:", err)
			return err
		}

		// New instance/trip specific seeders
		if err := SeedBusInstances(tx); err != nil {
			log.Println("Error seeding bus instances:", err)
			return err
		}

		if err := SeedBoardingPoints(tx); err != nil {
			log.Println("Error seeding boarding points:", err)
			return err
		}

		if err := SeedDroppingPoints(tx); err != nil {
			log.Println("Error seeding dropping points:", err)
			return err
		}

		if err := SeedFareTypes(tx); err != nil {
			log.Println("Error seeding fare types:", err)
			return err
		}

		// Configurations
		if err := SeedCancellationPolicies(tx); err != nil {
			log.Println("Error seeding cancellation policies:", err)
			return err
		}

		if err := SeedPricingRules(tx); err != nil {
			log.Println("Error seeding pricing rules:", err)
			return err
		}

		return nil
	})

	if err != nil {
		log.Println("Seeding Data Failed, err =", err)
		return err
	}

	log.Println("Seeding completed successfully")
	return nil
}
