package seed

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/lib/pq"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/gorm"
)

func SeedStations(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/stations.json")
	if err != nil {
		return err
	}
	var stations []models.Station
	if err := json.Unmarshal(bytes, &stations); err != nil {
		return err
	}
	for _, s := range stations {
		if err := tx.Where("code = ?", s.Code).FirstOrCreate(&s).Error; err != nil {
			return err
		}
	}
	return nil
}

// parseMinutes converts "HH:MM" to total minutes since midnight.
func parseMinutes(t string) int {
	var h, m int
	fmt.Sscanf(t, "%d:%d", &h, &m)
	return h*60 + m
}

func SeedTrains(tx *gorm.DB) error {
	bytes, err := os.ReadFile("data/trains.json")
	if err != nil {
		return err
	}

	var rawTrains []struct {
		TrainNumber string  `json:"train_number"`
		TrainName   string  `json:"train_name"`
		DaysOfWeek  []int32 `json:"days_of_week"`
		IsActive    bool    `json:"is_active"`
		Stops       []struct {
			StationCode string `json:"station_code"`
			Sequence    int    `json:"sequence"`
			Arrival     string `json:"arrival"`
			Departure   string `json:"departure"`
			DayOffset   int    `json:"day_offset"`
			Distance    int    `json:"distance"`
		} `json:"stops"`
	}

	if err := json.Unmarshal(bytes, &rawTrains); err != nil {
		return err
	}

	for _, r := range rawTrains {
		var origin, destination, depTime, arrTime string
		var durationMinutes int

		if len(r.Stops) > 0 {
			firstStop := r.Stops[0]
			lastStop := r.Stops[len(r.Stops)-1]

			origin = firstStop.StationCode
			depTime = firstStop.Departure

			destination = lastStop.StationCode
			arrTime = lastStop.Arrival

			// Calculate actual duration: handle day-offset overnight journeys
			depMins := parseMinutes(firstStop.Departure)
			arrMins := parseMinutes(lastStop.Arrival) + (lastStop.DayOffset * 24 * 60)
			durationMinutes = arrMins - depMins
			if durationMinutes < 0 {
				durationMinutes = 0
			}
		}

		train := models.Train{
			TrainNumber:        r.TrainNumber,
			TrainName:          r.TrainName,
			OriginStation:      origin,
			DestinationStation: destination,
			DepartureTime:      depTime,
			ArrivalTime:        arrTime,
			DurationMinutes:    durationMinutes,
			DaysOfWeek:         pq.Int32Array(r.DaysOfWeek),
			IsActive:           r.IsActive,
		}

		if err := tx.Where("train_number = ?", train.TrainNumber).
			Assign(models.Train{
				OriginStation:      origin,
				DestinationStation: destination,
				DepartureTime:      depTime,
				ArrivalTime:        arrTime,
				DurationMinutes:    durationMinutes,
			}).
			FirstOrCreate(&train).Error; err != nil {
			return err
		}

		for _, stop := range r.Stops {
			var station models.Station
			if err := tx.Where("code = ?", stop.StationCode).First(&station).Error; err != nil {
				log.Printf("Warning: Station %s not found, skipping stop", stop.StationCode)
				continue
			}
			trainStop := models.TrainStop{
				TrainID:       train.ID,
				StationID:     station.ID,
				StopSequence:  stop.Sequence,
				ArrivalTime:   stop.Arrival,
				DepartureTime: stop.Departure,
				DayOffset:     stop.DayOffset,
				DistanceKm:    stop.Distance,
			}
			tx.Where("train_id = ? AND stop_sequence = ?", train.ID, stop.Sequence).
				FirstOrCreate(&trainStop)
		}
	}
	return nil
}
