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

const searchCacheTTL = 2 * time.Minute

func SearchTrains(
	ctx context.Context,
	rdb *goredis.Client,
	origin, destination, class, dateStr string,
) ([]repository.SearchResult, error) {

	// Parse date
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD")
	}

	// Build cache key
	cacheKey := fmt.Sprintf("train:search:%s:%s:%s:%s", origin, destination, class, dateStr)

	// Try cache first
	cached, err := rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var results []repository.SearchResult
		if jsonErr := json.Unmarshal([]byte(cached), &results); jsonErr == nil {
			log.Printf("search cache hit: %s", cacheKey)
			return results, nil
		}
	}

	// Cache miss — query DB
	results, err := repository.SearchTrains(origin, destination, class, date)
	if err != nil {
		return nil, err
	}

	// Cache the results
	if data, jsonErr := json.Marshal(results); jsonErr == nil {
		_ = rdb.Set(ctx, cacheKey, data, searchCacheTTL).Err()
	}

	return results, nil
}

// GetScheduleDetail returns a single schedule with its train details.
func GetScheduleDetail(scheduleID string) (interface{}, error) {
	schedule, err := repository.GetScheduleByID(scheduleID)
	if err != nil {
		return nil, err
	}
	return schedule, nil
}

// GetSeatMap returns the berth map for a schedule and class.
// Also checks Redis for per-seat lock status and marks locked seats.
func GetSeatMap(
	ctx context.Context,
	rdb *goredis.Client,
	scheduleID, class string,
) (interface{}, error) {
	seats, err := repository.GetSeatsByScheduleAndClass(scheduleID, class)
	if err != nil {
		return nil, err
	}

	type SeatWithLock struct {
		ID         string  `json:"id"`
		SeatNumber string  `json:"seat_number"`
		Coach      string  `json:"coach"`
		Class      string  `json:"class"`
		BerthType  string  `json:"berth_type"`
		Status     string  `json:"status"`
		Price      float64 `json:"price"`
		IsLocked   bool    `json:"is_locked"`
	}

	result := make([]SeatWithLock, len(seats))
	for i, s := range seats {
		isLocked := false
		if s.Status == "AVAILABLE" {
			// Check Redis lock — display only, not for booking gate
			locked, _ := checkLockStatus(ctx, rdb, scheduleID, s.ID.String())
			isLocked = locked
		}
		result[i] = SeatWithLock{
			ID:         s.ID.String(),
			SeatNumber: s.SeatNumber,
			Coach:      s.Coach,
			Class:      s.Class,
			BerthType:  s.BerthType,
			Status:     s.Status,
			Price:      s.Price,
			IsLocked:   isLocked,
		}
	}
	return result, nil
}

// checkLockStatus is an internal helper that reads the Redis lock key.
func checkLockStatus(ctx context.Context, rdb *goredis.Client, scheduleID, seatID string) (bool, error) {
	key := fmt.Sprintf("seat:lock:train:%s:%s", scheduleID, seatID)
	_, err := rdb.Get(ctx, key).Result()
	if err == goredis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
