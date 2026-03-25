package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/repository"
	goredis "github.com/redis/go-redis/v9"
)

const trackingCacheTTL = 30 * time.Second

// TrackingResult is the live status response.
type TrackingResult struct {
	TrainNumber    string    `json:"train_number"`
	TrainName      string    `json:"train_name"`
	CurrentStation string    `json:"current_station"`
	NextStation    string    `json:"next_station"`
	DelayMinutes   int       `json:"delay_minutes"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Status         string    `json:"status"`
	Stale          bool      `json:"stale"`
	LastUpdated    time.Time `json:"last_updated"`
}

func GetLiveStatus(
	ctx context.Context,
	rdb *goredis.Client,
	scheduleID string,
) (*TrackingResult, error) {

	schedule, err := repository.GetScheduleByID(scheduleID)
	if err != nil {
		return nil, err
	}

	trainNumber := schedule.Train.TrainNumber
	cacheKey := fmt.Sprintf("train:status:%s", trainNumber)

	// Check cache
	cached, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var result TrackingResult
		if jsonErr := json.Unmarshal([]byte(cached), &result); jsonErr == nil {
			result.Stale = false
			return &result, nil
		}
	}

	result := simulateTracking(schedule.Train.TrainName, trainNumber, schedule.Status, schedule.DelayMinutes)

	// Store in Redis
	if data, jsonErr := json.Marshal(result); jsonErr == nil {
		_ = rdb.Set(ctx, cacheKey, data, trackingCacheTTL).Err()
	}

	return result, nil
}

func simulateTracking(trainName, trainNumber, status string, delayMinutes int) *TrackingResult {
	log.Printf("[tracking] simulated response for train %s", trainNumber)
	return &TrackingResult{
		TrainNumber:    trainNumber,
		TrainName:      trainName,
		CurrentStation: "En Route",
		NextStation:    "Next Stop",
		DelayMinutes:   delayMinutes,
		Latitude:       0,
		Longitude:      0,
		Status:         status,
		Stale:          false,
		LastUpdated:    time.Now(),
	}
}
