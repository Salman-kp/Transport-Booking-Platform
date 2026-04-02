package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/repository"
	goredis "github.com/redis/go-redis/v9"
)

const searchCacheTTL = 5 * time.Minute

func SearchTrains(
	ctx context.Context,
	rdb *goredis.Client,
	fromCode, toCode, dateStr, class string,
) ([]repository.SearchResult, error) {

	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}
	if class == "" {
		class = "SL"
	}

	// --- Redis cache check ---
	cacheKey := fmt.Sprintf("search:%s:%s:%s:%s", fromCode, toCode, dateStr, class)
	if cached, err := rdb.Get(ctx, cacheKey).Result(); err == nil {
		var results []repository.SearchResult
		if jsonErr := json.Unmarshal([]byte(cached), &results); jsonErr == nil {
			return results, nil
		}
	}

	results, err := repository.SearchTrains(fromCode, toCode, class, parsedDate)
	if err != nil {
		return nil, fmt.Errorf("failed to search trains: %w", err)
	}

	// Cache the result (ignore cache write errors)
	if data, marshalErr := json.Marshal(results); marshalErr == nil {
		rdb.Set(ctx, cacheKey, data, searchCacheTTL)
	}

	return results, nil
}

// GetScheduleDetail returns a single schedule with its train details and resolved stop times.
func GetScheduleDetail(scheduleID string) (interface{}, error) {
	schedule, err := repository.GetScheduleByID(scheduleID)
	if err != nil {
		return nil, err
	}

	type StopDetail struct {
		StationName     string `json:"station_name"`
		StationCode     string `json:"station_code"`
		StopSequence    int    `json:"stop_sequence"`
		ActualArrival   string `json:"actual_arrival"`
		ActualDeparture string `json:"actual_departure"`
		DistanceKm      int    `json:"distance_km"`
		DayOffset       int    `json:"day_offset"`
	}

	var stopDetails []StopDetail
	for _, stop := range schedule.Train.Stops {
		arrTime, _ := time.Parse("15:04", stop.ArrivalTime)
		depTime, _ := time.Parse("15:04", stop.DepartureTime)

		baseDate := schedule.ScheduleDate.AddDate(0, 0, stop.DayOffset)
		loc := schedule.ScheduleDate.Location()

		actualArr := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(),
			arrTime.Hour(), arrTime.Minute(), 0, 0, loc)
		actualDep := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(),
			depTime.Hour(), depTime.Minute(), 0, 0, loc)

		stopDetails = append(stopDetails, StopDetail{
			StationName:     stop.Station.Name,
			StationCode:     stop.Station.Code,
			StopSequence:    stop.StopSequence,
			ActualArrival:   actualArr.Format(time.RFC3339),
			ActualDeparture: actualDep.Format(time.RFC3339),
			DistanceKm:      stop.DistanceKm,
			DayOffset:       stop.DayOffset,
		})
	}

	result := map[string]interface{}{
		"schedule_id":   schedule.ID,
		"train_number":  schedule.Train.TrainNumber,
		"train_name":    schedule.Train.TrainName,
		"schedule_date": schedule.ScheduleDate.Format("2006-01-02"),
		"departure_at":  schedule.DepartureAt.Format(time.RFC3339),
		"arrival_at":    schedule.ArrivalAt.Format(time.RFC3339),
		"delay_minutes": schedule.DelayMinutes,
		"status":        schedule.Status,
		"available_sl":  schedule.AvailableSL,
		"available_3ac": schedule.Available3AC,
		"available_2ac": schedule.Available2AC,
		"available_1ac": schedule.Available1AC,
		"stops":         stopDetails,
	}

	return result, nil
}

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
