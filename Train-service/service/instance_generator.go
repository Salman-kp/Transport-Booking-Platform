package service

import (
	"fmt"
	"log"
	"time"

	"github.com/nabeel-mp/tripneo/train-service/db"
	"github.com/nabeel-mp/tripneo/train-service/models"
	"gorm.io/gorm"
)

type CoachConfig struct {
	// SL coaches
	SLCoaches int // e.g. 8 coaches named S1..S8
	SLBerths  []BerthConfig

	// 3AC coaches
	ThreeACCoaches int
	ThreeACBerths  []BerthConfig

	// 2AC coaches
	TwoACCoaches int
	TwoACBerths  []BerthConfig

	// 1AC coaches
	OneACCoaches int
	OneACBerths  []BerthConfig
}

// BerthConfig defines one berth slot in a coach.
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

func GenerateInstancesForDays(days int) error {
	log.Printf("[instance-gen] Starting generation for next %d days", days)

	// Fetch all active train templates
	var trains []models.Train
	if err := db.DB.Where("is_active = true").Find(&trains).Error; err != nil {
		return fmt.Errorf("failed to fetch trains: %w", err)
	}
	log.Printf("[instance-gen] Found %d active train templates", len(trains))

	today := time.Now().Truncate(24 * time.Hour)
	coachCfg := defaultCoachConfig()

	totalSchedules := 0
	totalInventory := 0

	for _, train := range trains {
		for d := 0; d < days; d++ {
			targetDate := today.Add(time.Duration(d) * 24 * time.Hour)

			// Check if this train runs on this day of week
			// time.Weekday(): Sunday=0, Monday=1, ..., Saturday=6
			// Our DaysOfWeek: 1=Monday, 7=Sunday (ISO 8601)
			isoWeekday := int(targetDate.Weekday())
			if isoWeekday == 0 {
				isoWeekday = 7 // Sunday = 7 in ISO
			}
			if !containsDay([]int32(train.DaysOfWeek), int32(isoWeekday)) {
				continue // This train doesn't run on this day
			}

			// Check if schedule already exists (idempotency)
			var existing models.TrainSchedule
			err := db.DB.
				Where("train_id = ? AND schedule_date = ?", train.ID, targetDate).
				First(&existing).Error
			if err == nil {
				continue // Already generated, skip
			}

			// Parse departure and arrival times
			depHour, depMin := parseTime(train.DepartureTime)
			arrHour, arrMin := parseTime(train.ArrivalTime)

			departureAt := time.Date(
				targetDate.Year(), targetDate.Month(), targetDate.Day(),
				depHour, depMin, 0, 0, time.Local,
			)

			// Arrival might be next day (e.g. 20:30 depart → 05:30 arrive)
			arrivalBase := targetDate
			if arrHour < depHour || (arrHour == depHour && arrMin < depMin) {
				arrivalBase = targetDate.Add(24 * time.Hour) // next day
			}
			arrivalAt := time.Date(
				arrivalBase.Year(), arrivalBase.Month(), arrivalBase.Day(),
				arrHour, arrMin, 0, 0, time.Local,
			)

			// Count berths per class for the schedule
			slCount := coachCfg.SLCoaches * len(coachCfg.SLBerths)
			threeACCount := coachCfg.ThreeACCoaches * len(coachCfg.ThreeACBerths)
			twoACCount := coachCfg.TwoACCoaches * len(coachCfg.TwoACBerths)
			oneACCount := coachCfg.OneACCoaches * len(coachCfg.OneACBerths)

			txErr := db.DB.Transaction(func(tx *gorm.DB) error {
				// Create TrainSchedule
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
					return fmt.Errorf("create schedule failed: %w", err)
				}

				// Generate inventory for each class
				var allBerths []models.TrainInventory

				// SL coaches: S1, S2, S3...
				for c := 1; c <= coachCfg.SLCoaches; c++ {
					coach := fmt.Sprintf("S%d", c)
					for _, b := range coachCfg.SLBerths {
						allBerths = append(allBerths, models.TrainInventory{
							TrainScheduleID: schedule.ID,
							SeatNumber:      b.SeatNumber,
							Coach:           coach,
							Class:           "SL",
							BerthType:       b.BerthType,
							Status:          "AVAILABLE",
							Price:           b.Price,
							WholesalePrice:  b.Wholesale,
						})
					}
				}

				// 3AC coaches: B1, B2...
				for c := 1; c <= coachCfg.ThreeACCoaches; c++ {
					coach := fmt.Sprintf("B%d", c)
					for _, b := range coachCfg.ThreeACBerths {
						allBerths = append(allBerths, models.TrainInventory{
							TrainScheduleID: schedule.ID,
							SeatNumber:      b.SeatNumber,
							Coach:           coach,
							Class:           "3AC",
							BerthType:       b.BerthType,
							Status:          "AVAILABLE",
							Price:           b.Price,
							WholesalePrice:  b.Wholesale,
						})
					}
				}

				// 2AC coaches: A1, A2...
				for c := 1; c <= coachCfg.TwoACCoaches; c++ {
					coach := fmt.Sprintf("A%d", c)
					for _, b := range coachCfg.TwoACBerths {
						allBerths = append(allBerths, models.TrainInventory{
							TrainScheduleID: schedule.ID,
							SeatNumber:      b.SeatNumber,
							Coach:           coach,
							Class:           "2AC",
							BerthType:       b.BerthType,
							Status:          "AVAILABLE",
							Price:           b.Price,
							WholesalePrice:  b.Wholesale,
						})
					}
				}

				// 1AC coaches: H1...
				for c := 1; c <= coachCfg.OneACCoaches; c++ {
					coach := fmt.Sprintf("H%d", c)
					for _, b := range coachCfg.OneACBerths {
						allBerths = append(allBerths, models.TrainInventory{
							TrainScheduleID: schedule.ID,
							SeatNumber:      b.SeatNumber,
							Coach:           coach,
							Class:           "1AC",
							BerthType:       b.BerthType,
							Status:          "AVAILABLE",
							Price:           b.Price,
							WholesalePrice:  b.Wholesale,
						})
					}
				}

				if err := tx.Create(&allBerths).Error; err != nil {
					return fmt.Errorf("create inventory failed: %w", err)
				}

				totalSchedules++
				totalInventory += len(allBerths)
				log.Printf("[instance-gen] ✓ %s %s → %s on %s (%d berths)",
					train.TrainNumber, train.OriginStation,
					train.DestinationStation,
					targetDate.Format("2006-01-02"),
					len(allBerths),
				)
				return nil
			})

			if txErr != nil {
				log.Printf("[instance-gen] ERROR for train %s on %s: %v",
					train.TrainNumber, targetDate.Format("2006-01-02"), txErr)
			}
		}
	}

	log.Printf("[instance-gen] Complete. Schedules: %d  Inventory rows: %d",
		totalSchedules, totalInventory)
	return nil
}

// RunInstanceGeneratorWorker is the background goroutine.
// Generates instances for the next 30 days at startup,
// then re-runs every day at 2AM to keep the rolling window fresh.
func RunInstanceGeneratorWorker() {
	log.Println("[instance-gen] Worker started")

	// Run immediately at startup
	if err := GenerateInstancesForDays(30); err != nil {
		log.Printf("[instance-gen] Startup generation error: %v", err)
	}

	// Then run every day at 2AM
	for {
		now := time.Now()
		// Next 2AM
		next2AM := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
		if now.After(next2AM) {
			next2AM = next2AM.Add(24 * time.Hour)
		}
		sleepDuration := time.Until(next2AM)
		log.Printf("[instance-gen] Next run in %v (at %s)",
			sleepDuration.Round(time.Minute),
			next2AM.Format("2006-01-02 02:00"),
		)
		time.Sleep(sleepDuration)

		if err := GenerateInstancesForDays(30); err != nil {
			log.Printf("[instance-gen] Daily generation error: %v", err)
		}
	}
}

// parseTime splits "20:30" into hour=20, min=30.
func parseTime(t string) (int, int) {
	var h, m int
	fmt.Sscanf(t, "%d:%d", &h, &m)
	return h, m
}

// containsDay checks if a day number exists in the DaysOfWeek array.
func containsDay(days []int32, day int32) bool {
	for _, d := range days {
		if d == day {
			return true
		}
	}
	return false
}
