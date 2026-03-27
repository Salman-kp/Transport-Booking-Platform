package seed

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

func SeedBusInstances(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/bus_instances.json")
	if err != nil {
		return err
	}
	var raw []struct {
		BusNumber           string    `json:"bus_number"`
		TravelDate          string    `json:"travel_date"`
		DepartureAt         time.Time `json:"departure_at"`
		ArrivalAt           time.Time `json:"arrival_at"`
		Status              string    `json:"status"`
		BasePriceSeater         float64 `json:"base_price_seater"`
		BasePriceSemiSleeper    float64 `json:"base_price_semi_sleeper"`
		BasePriceSleeper        float64 `json:"base_price_sleeper"`
		CurrentPriceSeater      float64 `json:"current_price_seater"`
		CurrentPriceSemiSleeper float64 `json:"current_price_semi_sleeper"`
		CurrentPriceSleeper     float64 `json:"current_price_sleeper"`
	}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return err
	}

	for _, r := range raw {
		var templateBus model.Bus
		// Deep query to retrieve parsing Layout parameters immediately
		if err := tx.Preload("BusType").Where("bus_number = ?", r.BusNumber).First(&templateBus).Error; err != nil {
			continue
		}
		tDate, _ := time.Parse("2006-01-02", r.TravelDate)
		
		inst := model.BusInstance{
			BusID: templateBus.ID,
			TravelDate: tDate,
			DepartureAt: r.DepartureAt,
			ArrivalAt: r.ArrivalAt,
			Status: r.Status,
			BasePriceSeater: r.BasePriceSeater,
			BasePriceSemiSleeper: r.BasePriceSemiSleeper,
			BasePriceSleeper: r.BasePriceSleeper,
			CurrentPriceSeater: r.CurrentPriceSeater,
			CurrentPriceSemiSleeper: r.CurrentPriceSemiSleeper,
			CurrentPriceSleeper: r.CurrentPriceSleeper,
		}

		var existing model.BusInstance
		if err := tx.Where("bus_id = ? AND travel_date = ?", templateBus.ID, tDate).First(&existing).Error; err != nil {
			// Did not exist, evaluate and create immediately!
			if err := tx.Create(&inst).Error; err != nil {
				return err
			}
			// Dynamically compute exact individual Seats inside PG natively without JSON fatigue
			if err := ComputationallyMapSeats(tx, inst.ID, templateBus.BusType.SeatLayout); err != nil {
				return err
			}
		}
	}
	return nil
}

func SeedBoardingPoints(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/boarding_points.json")
	if err != nil {
		return err
	}
	var raw []struct {
		BusNumber string `json:"bus_number"`
		TravelDate string `json:"travel_date"`
		StopName string `json:"stop_name"`
		PickupTime time.Time `json:"pickup_time"`
		SequenceOrder int `json:"sequence_order"`
		Landmark string `json:"landmark"`
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
		var stop model.BusStop
		if err := tx.Where("name = ?", r.StopName).First(&stop).Error; err != nil {
			continue
		}

		bp := model.BoardingPoint{
			BusInstanceID: inst.ID,
			BusStopID: stop.ID,
			PickupTime: r.PickupTime,
			SequenceOrder: r.SequenceOrder,
			Landmark: r.Landmark,
		}
		tx.Where("bus_instance_id = ? AND bus_stop_id = ?", inst.ID, stop.ID).FirstOrCreate(&bp)
	}
	return nil
}

func SeedDroppingPoints(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/dropping_points.json")
	if err != nil {
		return err
	}
	var raw []struct {
		BusNumber string `json:"bus_number"`
		TravelDate string `json:"travel_date"`
		StopName string `json:"stop_name"`
		DropTime time.Time `json:"drop_time"`
		SequenceOrder int `json:"sequence_order"`
		Landmark string `json:"landmark"`
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
		var stop model.BusStop
		if err := tx.Where("name = ?", r.StopName).First(&stop).Error; err != nil {
			continue
		}

		dp := model.DroppingPoint{
			BusInstanceID: inst.ID,
			BusStopID: stop.ID,
			DropTime: r.DropTime,
			SequenceOrder: r.SequenceOrder,
			Landmark: r.Landmark,
		}
		tx.Where("bus_instance_id = ? AND bus_stop_id = ?", inst.ID, stop.ID).FirstOrCreate(&dp)
	}
	return nil
}
