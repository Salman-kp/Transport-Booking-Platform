package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/repository"
	"github.com/nabeel-mp/tripneo/train-service/utils"
	goredis "github.com/redis/go-redis/v9"
)

const trackingCacheTTL = 1 * time.Minute

type TrackingResult struct {
	TrainNumber    string    `json:"train_number"`
	CurrentStation string    `json:"current_station"`
	NextStation    string    `json:"next_station"`
	DelayMinutes   int       `json:"delay_minutes"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	LastUpdated    time.Time `json:"last_updated"`
	Source         string    `json:"source"`
}

func GetLiveStatus(ctx context.Context, rdb *goredis.Client, scheduleID string) (*TrackingResult, error) {
	schedule, err := repository.GetScheduleByID(scheduleID)
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("live:%s", schedule.Train.TrainNumber)

	// 1. Try Redis Cache
	if cached, err := rdb.Get(ctx, cacheKey).Result(); err == nil {
		var res TrackingResult
		json.Unmarshal([]byte(cached), &res)
		return &res, nil
	}

	// 2. Try External API
	externalData, err := utils.FetchLiveStatusFromExternalAPI(schedule.Train.TrainNumber)
	var finalResult TrackingResult

	if err == nil {
		finalResult = TrackingResult{
			TrainNumber:    schedule.Train.TrainNumber,
			CurrentStation: externalData.CurrentStation,
			NextStation:    externalData.NextStation,
			DelayMinutes:   externalData.DelayMinutes,
			Latitude:       externalData.Lat,
			Longitude:      externalData.Lon,
			LastUpdated:    time.Now(),
			Source:         "API",
		}
	} else {
		// 3. Fallback to Local Simulation if API fails
		finalResult = TrackingResult{
			TrainNumber:    schedule.Train.TrainNumber,
			CurrentStation: "In Transit",
			DelayMinutes:   schedule.DelayMinutes,
			LastUpdated:    time.Now(),
			Source:         "Local_Sim",
		}
	}

	// Store in Cache
	data, _ := json.Marshal(finalResult)
	rdb.Set(ctx, cacheKey, data, trackingCacheTTL)

	return &finalResult, nil
}
