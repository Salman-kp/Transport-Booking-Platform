package seed

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

func SeedFareTypes(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/fare_types.json")
	if err != nil {
		return err
	}
	var raw []struct {
		BusNumber string `json:"bus_number"`
		TravelDate string `json:"travel_date"`
		Name string `json:"name"`
		SeatType string `json:"seat_type"`
		Price float64 `json:"price"`
		IsRefundable bool `json:"is_refundable"`
		CancellationFee float64 `json:"cancellation_fee"`
		DateChangeFee float64 `json:"date_change_fee"`
	}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return err
	}

	for _, r := range raw {
		var bus model.Bus
		if err := tx.Where("bus_number = ?", r.BusNumber).First(&bus).Error; err != nil {
			continue
		}
		var inst model.BusInstance
		tDate, _ := time.Parse("2006-01-02", r.TravelDate)
		if err := tx.Where("bus_id = ? AND travel_date = ?", bus.ID, tDate).First(&inst).Error; err != nil {
			continue
		}

		ft := model.FareType{
			BusInstanceID: inst.ID,
			SeatType: r.SeatType,
			Name: r.Name,
			Price: r.Price,
			IsRefundable: r.IsRefundable,
			CancellationFee: r.CancellationFee,
			DateChangeFee: r.DateChangeFee,
			SeatsAvailable: 0, // Should realistically bind dynamically based on available, defaults to 0 as proxy in prototype 
		}
		tx.Where("bus_instance_id = ? AND name = ? AND seat_type = ?", inst.ID, r.Name, r.SeatType).FirstOrCreate(&ft)
	}
	return nil
}
