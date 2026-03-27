package service

import (
	"context"
	"fmt"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/repository"
	goredis "github.com/redis/go-redis/v9"
)

const searchCacheTTL = 2 * time.Minute

func SearchTrains(ctx context.Context, rdb *goredis.Client, fromCode, toCode, dateStr, class string) ([]repository.SearchResult, error) {
	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	if class == "" {
		class = "SL"
	}

	// Call the repository function
	results, err := repository.SearchTrains(fromCode, toCode, class, parsedDate)
	if err != nil {
		return nil, fmt.Errorf("failed to search trains: %w", err)
	}

	return results, nil
}

// GetScheduleDetail returns a single schedule with its train details.
func GetScheduleDetail(scheduleID string) (interface{}, error) {
	schedule, err := repository.GetScheduleByID(scheduleID)
	if err != nil {
		return nil, err
	}

	// Define a custom struct to format the stop details nicely for the frontend
	type StopDetail struct {
		StationName     string `json:"station_name"`
		StationCode     string `json:"station_code"`
		StopSequence    int    `json:"stop_sequence"`
		ActualArrival   string `json:"actual_arrival"`
		ActualDeparture string `json:"actual_departure"`
		DistanceKm      int    `json:"distance_km"`
	}

	var stopDetails []StopDetail

	for _, stop := range schedule.Train.Stops {
		arrTime, _ := time.Parse("15:04", stop.ArrivalTime)
		depTime, _ := time.Parse("15:04", stop.DepartureTime)

		baseDate := schedule.ScheduleDate.AddDate(0, 0, stop.DayOffset)

		actualArr := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), arrTime.Hour(), arrTime.Minute(), 0, 0, schedule.ScheduleDate.Location())
		actualDep := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), depTime.Hour(), depTime.Minute(), 0, 0, schedule.ScheduleDate.Location())

		stopDetails = append(stopDetails, StopDetail{
			StationName:     stop.Station.Name,
			StationCode:     stop.Station.Code,
			StopSequence:    stop.StopSequence,
			ActualArrival:   actualArr.Format(time.RFC3339),
			ActualDeparture: actualDep.Format(time.RFC3339),
			DistanceKm:      stop.DistanceKm,
		})
	}

	// Build the final detailed response
	result := map[string]interface{}{
		"schedule_id":   schedule.ID,
		"train_number":  schedule.Train.TrainNumber,
		"train_name":    schedule.Train.TrainName,
		"schedule_date": schedule.ScheduleDate.Format("2006-01-02"),
		"status":        schedule.Status,
		"delay_minutes": schedule.DelayMinutes,
		"available_sl":  schedule.AvailableSL,
		"available_3ac": schedule.Available3AC,
		"available_2ac": schedule.Available2AC,
		"available_1ac": schedule.Available1AC,
		"stops":         stopDetails, // This will now show the clean, calculated array!
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
