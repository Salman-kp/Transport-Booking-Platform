package jobs

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/gorm"
)

type CoachConfig struct {
	SLCoaches      int
	SLBerths       []BerthConfig
	ThreeACCoaches int
	ThreeACBerths  []BerthConfig
	TwoACCoaches   int
	TwoACBerths    []BerthConfig
	OneACCoaches   int
	OneACBerths    []BerthConfig
}

type BerthConfig struct {
	SeatNumber string
	BerthType  string // LOWER | MIDDLE | UPPER | SIDE_LOWER | SIDE_UPPER
	Price      float64
	Wholesale  float64
}

func defaultCoachConfig() CoachConfig {
	slBerths := []BerthConfig{
		{SeatNumber: "1", BerthType: "LOWER", Price: 750, Wholesale: 600},
		{SeatNumber: "2", BerthType: "MIDDLE", Price: 720, Wholesale: 575},
		{SeatNumber: "3", BerthType: "UPPER", Price: 700, Wholesale: 555},
		{SeatNumber: "4", BerthType: "LOWER", Price: 750, Wholesale: 600},
		{SeatNumber: "5", BerthType: "MIDDLE", Price: 720, Wholesale: 575},
		{SeatNumber: "6", BerthType: "UPPER", Price: 700, Wholesale: 555},
		{SeatNumber: "7", BerthType: "SIDE_LOWER", Price: 680, Wholesale: 540},
		{SeatNumber: "8", BerthType: "SIDE_UPPER", Price: 660, Wholesale: 525},
	}

	threACBerths := []BerthConfig{
		{SeatNumber: "1", BerthType: "LOWER", Price: 1450, Wholesale: 1200},
		{SeatNumber: "2", BerthType: "MIDDLE", Price: 1400, Wholesale: 1150},
		{SeatNumber: "3", BerthType: "UPPER", Price: 1350, Wholesale: 1100},
		{SeatNumber: "4", BerthType: "LOWER", Price: 1450, Wholesale: 1200},
		{SeatNumber: "5", BerthType: "MIDDLE", Price: 1400, Wholesale: 1150},
		{SeatNumber: "6", BerthType: "UPPER", Price: 1350, Wholesale: 1100},
		{SeatNumber: "7", BerthType: "SIDE_LOWER", Price: 1300, Wholesale: 1050},
		{SeatNumber: "8", BerthType: "SIDE_UPPER", Price: 1280, Wholesale: 1030},
	}

	twoACBerths := []BerthConfig{
		{SeatNumber: "1", BerthType: "LOWER", Price: 2100, Wholesale: 1800},
		{SeatNumber: "2", BerthType: "UPPER", Price: 2000, Wholesale: 1700},
		{SeatNumber: "3", BerthType: "LOWER", Price: 2100, Wholesale: 1800},
		{SeatNumber: "4", BerthType: "UPPER", Price: 2000, Wholesale: 1700},
	}

	oneACBerths := []BerthConfig{
		{SeatNumber: "1", BerthType: "LOWER", Price: 3500, Wholesale: 3000},
		{SeatNumber: "2", BerthType: "UPPER", Price: 3300, Wholesale: 2800},
	}

	return CoachConfig{
		SLCoaches:      3,
		SLBerths:       slBerths,
		ThreeACCoaches: 2,
		ThreeACBerths:  threACBerths,
		TwoACCoaches:   1,
		TwoACBerths:    twoACBerths,
		OneACCoaches:   1,
		OneACBerths:    oneACBerths,
	}
}

// GenerateUpcomingInventory matches the call in main.go
func GenerateUpcomingInventory(database *gorm.DB, days int) error {
	log.Printf("[instance-gen] Starting generation for next %d days", days)

	var trains []models.Train
	// Preload Stops and Station names for logging/logic
	if err := database.Preload("Stops", func(db *gorm.DB) *gorm.DB {
		return db.Order("stop_sequence ASC").Preload("Station")
	}).Where("is_active = true").Find(&trains).Error; err != nil {
		return fmt.Errorf("failed to fetch trains: %w", err)
	}

	today := time.Now().Truncate(24 * time.Hour)
	coachCfg := defaultCoachConfig()

	totalSchedules := 0
	totalInventory := 0

	for _, train := range trains {
		if len(train.Stops) < 2 {
			log.Printf("[instance-gen] Skipping Train %s: Not enough stops configured", train.TrainNumber)
			continue
		}

		firstStop := train.Stops[0]
		lastStop := train.Stops[len(train.Stops)-1]

		for d := 0; d < days; d++ {
			targetDate := today.Add(time.Duration(d) * 24 * time.Hour)

			isoWeekday := int(targetDate.Weekday())
			if isoWeekday == 0 {
				isoWeekday = 7
			}
			if !containsDay([]int32(train.DaysOfWeek), int32(isoWeekday)) {
				continue
			}

			// Check if schedule already exists
			var existing models.TrainSchedule
			err := database.Where("train_id = ? AND schedule_date = ?", train.ID, targetDate).First(&existing).Error
			if err == nil {
				continue
			}

			// Departure from first stop on target date
			depHour, depMin := parseTime(firstStop.DepartureTime)
			departureAt := time.Date(
				targetDate.Year(), targetDate.Month(), targetDate.Day(),
				depHour, depMin, 0, 0, time.Local,
			)

			// Arrival at last stop using DayOffset
			arrHour, arrMin := parseTime(lastStop.ArrivalTime)
			arrivalBase := targetDate.Add(time.Duration(lastStop.DayOffset) * 24 * time.Hour)
			arrivalAt := time.Date(
				arrivalBase.Year(), arrivalBase.Month(), arrivalBase.Day(),
				arrHour, arrMin, 0, 0, time.Local,
			)

			// Total berth counts
			slCount := coachCfg.SLCoaches * len(coachCfg.SLBerths)
			threeACCount := coachCfg.ThreeACCoaches * len(coachCfg.ThreeACBerths)
			twoACCount := coachCfg.TwoACCoaches * len(coachCfg.TwoACBerths)
			oneACCount := coachCfg.OneACCoaches * len(coachCfg.OneACBerths)

			txErr := database.Transaction(func(tx *gorm.DB) error {
				schedule := models.TrainSchedule{
					TrainID:      train.ID,
					ScheduleDate: targetDate,
					DepartureAt:  departureAt,
					ArrivalAt:    arrivalAt,
					Status:       "SCHEDULED",
					DelayMinutes: 0,
					AvailableSL:  slCount,
					Available3AC: threeACCount,
					Available2AC: twoACCount,
					Available1AC: oneACCount,
				}
				if err := tx.Create(&schedule).Error; err != nil {
					return err
				}

				var allBerths []models.TrainInventory

				// SL
				for c := 1; c <= coachCfg.SLCoaches; c++ {
					coach := fmt.Sprintf("S%d", c)
					for _, b := range coachCfg.SLBerths {
						allBerths = append(allBerths, buildInventory(schedule.ID, coach, "SL", b))
					}
				}
				// 3AC
				for c := 1; c <= coachCfg.ThreeACCoaches; c++ {
					coach := fmt.Sprintf("B%d", c)
					for _, b := range coachCfg.ThreeACBerths {
						allBerths = append(allBerths, buildInventory(schedule.ID, coach, "3AC", b))
					}
				}
				// 2AC
				for c := 1; c <= coachCfg.TwoACCoaches; c++ {
					coach := fmt.Sprintf("A%d", c)
					for _, b := range coachCfg.TwoACBerths {
						allBerths = append(allBerths, buildInventory(schedule.ID, coach, "2AC", b))
					}
				}
				// 1AC
				for c := 1; c <= coachCfg.OneACCoaches; c++ {
					coach := fmt.Sprintf("H%d", c)
					for _, b := range coachCfg.OneACBerths {
						allBerths = append(allBerths, buildInventory(schedule.ID, coach, "1AC", b))
					}
				}

				if err := tx.Create(&allBerths).Error; err != nil {
					return err
				}

				totalSchedules++
				totalInventory += len(allBerths)
				log.Printf("[instance-gen] Created %s (%s -> %s) for %s",
					train.TrainNumber, firstStop.Station.Code, lastStop.Station.Code, targetDate.Format("2006-01-02"))
				return nil
			})

			if txErr != nil {
				log.Printf("[instance-gen] ERROR for train %s: %v", train.TrainNumber, txErr)
			}
		}
	}
	log.Printf("[instance-gen] Completed. Schedules: %d, Berths: %d", totalSchedules, totalInventory)
	return nil
}

func buildInventory(scheduleID uuid.UUID, coach string, class string, b BerthConfig) models.TrainInventory {
	return models.TrainInventory{
		TrainScheduleID: scheduleID,
		SeatNumber:      b.SeatNumber,
		Coach:           coach,
		Class:           class,
		BerthType:       b.BerthType,
		Status:          "AVAILABLE",
		Price:           b.Price,
		WholesalePrice:  b.Wholesale,
	}
}

func parseTime(t string) (int, int) {
	var h, m int
	fmt.Sscanf(t, "%d:%d", &h, &m)
	return h, m
}

func containsDay(days []int32, day int32) bool {
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}
