package seed

import (
	"encoding/json"
	"log"
	"os"
	"sort"
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
		BusNumber               string    `json:"bus_number"`
		TravelDate              string    `json:"travel_date"`
		DepartureAt             time.Time `json:"departure_at"`
		ArrivalAt               time.Time `json:"arrival_at"`
		Status                  string    `json:"status"`
		DelayMinutes            int       `json:"delay_minutes"`
		AvailableSeater         int       `json:"available_seater"`
		AvailableSemiSleeper    int       `json:"available_semi_sleeper"`
		AvailableSleeper        int       `json:"available_sleeper"`
		BasePriceSeater         float64   `json:"base_price_seater"`
		BasePriceSemiSleeper    float64   `json:"base_price_semi_sleeper"`
		BasePriceSleeper        float64   `json:"base_price_sleeper"`
		CurrentPriceSeater      float64   `json:"current_price_seater"`
		CurrentPriceSemiSleeper float64   `json:"current_price_semi_sleeper"`
		CurrentPriceSleeper     float64   `json:"current_price_sleeper"`
	}
	if err := json.Unmarshal(bytes, &raw); err != nil {
		return err
	}

	for _, r := range raw {
		// 0. Basic Validation
		if r.AvailableSeater < 0 || r.AvailableSemiSleeper < 0 || r.AvailableSleeper < 0 ||
			r.BasePriceSeater < 0 || r.BasePriceSemiSleeper < 0 || r.BasePriceSleeper < 0 {
			log.Printf("[seed] skipping invalid instance: negative availability or price for %s\n", r.BusNumber)
			continue
		}

		var templateBus model.Bus
		if err := tx.Preload("BusType").Where("bus_number = ?", r.BusNumber).First(&templateBus).Error; err != nil {
			log.Printf("[seed] skipping instance: template bus %s not found\n", r.BusNumber)
			continue
		}
		tDate, _ := time.Parse("2006-01-02", r.TravelDate)

		inst := model.BusInstance{
			BusID:                   templateBus.ID,
			TravelDate:              tDate,
			DepartureAt:             r.DepartureAt,
			ArrivalAt:               r.ArrivalAt,
			Status:                  r.Status,
			DelayMinutes:            r.DelayMinutes,
			AvailableSeater:         r.AvailableSeater,
			AvailableSemiSleeper:    r.AvailableSemiSleeper,
			AvailableSleeper:        r.AvailableSleeper,
			BasePriceSeater:         r.BasePriceSeater,
			BasePriceSemiSleeper:    r.BasePriceSemiSleeper,
			BasePriceSleeper:        r.BasePriceSleeper,
			CurrentPriceSeater:      r.CurrentPriceSeater,
			CurrentPriceSemiSleeper: r.CurrentPriceSemiSleeper,
			CurrentPriceSleeper:     r.CurrentPriceSleeper,
		}

		var existing model.BusInstance
		if err := tx.Where("bus_id = ? AND travel_date = ?", templateBus.ID, tDate).First(&existing).Error; err != nil {
			if err := tx.Create(&inst).Error; err != nil {
				log.Printf("[seed] error creating instance for %s on %s: %v\n", r.BusNumber, r.TravelDate, err)
				return err
			}
			// Dynamically generate seats
			if err := ComputationallyMapSeats(tx, inst.ID, templateBus.BusType.SeatLayout); err != nil {
				log.Printf("[seed] error generating seats for instance %s: %v\n", inst.ID, err)
				return err
			}
		}
	}
	log.Println("✅ Bus instance seeding completed")
	return nil
}


type rawRoutePoint struct {
	StopName      string
	City          string
	Time          time.Time
	SequenceOrder int
	Landmark      string
	IsBoarding    bool
}

func SeedBoardingPoints(tx *gorm.DB) error {
	return seedRoutePoints(tx, true)
}

func SeedDroppingPoints(tx *gorm.DB) error {
	return seedRoutePoints(tx, false)
}

func seedRoutePoints(tx *gorm.DB, isBoardingMode bool) error {
	bBytes, err := os.ReadFile("data/boarding_points.json")
	if err != nil {
		return err
	}
	dBytes, err := os.ReadFile("data/dropping_points.json")
	if err != nil {
		return err
	}

	var bRaw []struct {
		BusNumber     string    `json:"bus_number"`
		TravelDate    string    `json:"travel_date"`
		StopName      string    `json:"stop_name"`
		City          string    `json:"city"`
		PickupTime    time.Time `json:"pickup_time"`
		SequenceOrder int       `json:"sequence_order"`
		Landmark      string    `json:"landmark"`
	}
	var dRaw []struct {
		BusNumber     string    `json:"bus_number"`
		TravelDate    string    `json:"travel_date"`
		StopName      string    `json:"stop_name"`
		City          string    `json:"city"`
		DropTime      time.Time `json:"drop_time"`
		SequenceOrder int       `json:"sequence_order"`
		Landmark      string    `json:"landmark"`
	}

	if err := json.Unmarshal(bBytes, &bRaw); err != nil {
		return err
	}
	if err := json.Unmarshal(dBytes, &dRaw); err != nil {
		return err
	}

	// Group points by BusInstance (BusNumber + TravelDate)
	type instanceKey struct {
		BusNumber  string
		TravelDate string
	}
	groups := make(map[instanceKey]*struct {
		boarding []rawRoutePoint
		dropping []rawRoutePoint
	})

	for _, r := range bRaw {
		key := instanceKey{r.BusNumber, r.TravelDate}
		if groups[key] == nil {
			groups[key] = &struct {
				boarding []rawRoutePoint
				dropping []rawRoutePoint
			}{}
		}
		groups[key].boarding = append(groups[key].boarding, rawRoutePoint{
			StopName:      r.StopName,
			City:          r.City,
			Time:          r.PickupTime,
			SequenceOrder: r.SequenceOrder,
			Landmark:      r.Landmark,
			IsBoarding:    true,
		})
	}
	for _, r := range dRaw {
		key := instanceKey{r.BusNumber, r.TravelDate}
		if groups[key] == nil {
			groups[key] = &struct {
				boarding []rawRoutePoint
				dropping []rawRoutePoint
			}{}
		}
		groups[key].dropping = append(groups[key].dropping, rawRoutePoint{
			StopName:      r.StopName,
			City:          r.City,
			Time:          r.DropTime,
			SequenceOrder: r.SequenceOrder,
			Landmark:      r.Landmark,
			IsBoarding:    false,
		})
	}

	for key, g := range groups {
		// 1. Fetch Bus and Instance
		var bus model.Bus
		if err := tx.Where("bus_number = ?", key.BusNumber).First(&bus).Error; err != nil {
			continue
		}
		tDate, _ := time.Parse("2006-01-02", key.TravelDate)
		var inst model.BusInstance
		if err := tx.Where("bus_id = ? AND travel_date = ?", bus.ID, tDate).First(&inst).Error; err != nil {
			continue
		}

		// 2. Sort points by sequence
		sort.Slice(g.boarding, func(i, j int) bool { return g.boarding[i].SequenceOrder < g.boarding[j].SequenceOrder })
		sort.Slice(g.dropping, func(i, j int) bool { return g.dropping[i].SequenceOrder < g.dropping[j].SequenceOrder })

		// 3. Validation Logic
		if len(g.boarding) == 0 || len(g.dropping) == 0 {
			log.Printf("[seed] skipping %s (%s): missing boarding or dropping points\n", key.BusNumber, key.TravelDate)
			continue
		}

		// Sequence Integrity: Strictly increasing and no overlaps
		maxBoardingSeq := g.boarding[len(g.boarding)-1].SequenceOrder
		minDroppingSeq := g.dropping[0].SequenceOrder

		if maxBoardingSeq >= minDroppingSeq {
			log.Printf("[seed] skipping %s (%s): sequence overlap! boarding max (%d) >= dropping min (%d)\n",
				key.BusNumber, key.TravelDate, maxBoardingSeq, minDroppingSeq)
			continue
		}

		// Strictly increasing check and duplicate stop check
		isValid := true
		stopMap := make(map[string]bool)

		for i, p := range g.boarding {
			if i > 0 && p.SequenceOrder <= g.boarding[i-1].SequenceOrder {
				log.Printf("[seed] skipping %s (%s): non-strictly increasing boarding sequence at order %d\n", key.BusNumber, key.TravelDate, p.SequenceOrder)
				isValid = false
				break
			}
			stopKey := p.StopName + "|" + p.City
			if stopMap[stopKey] {
				log.Printf("[seed] skipping %s (%s): duplicate stop %s\n", key.BusNumber, key.TravelDate, p.StopName)
				isValid = false
				break
			}
			stopMap[stopKey] = true
		}
		if !isValid {
			continue
		}

		for i, p := range g.dropping {
			if i > 0 && p.SequenceOrder <= g.dropping[i-1].SequenceOrder {
				log.Printf("[seed] skipping %s (%s): non-strictly increasing dropping sequence at order %d\n", key.BusNumber, key.TravelDate, p.SequenceOrder)
				isValid = false
				break
			}
			stopKey := p.StopName + "|" + p.City
			if stopMap[stopKey] {
				log.Printf("[seed] skipping %s (%s): duplicate stop %s\n", key.BusNumber, key.TravelDate, p.StopName)
				isValid = false
				break
			}
			stopMap[stopKey] = true
		}
		if !isValid {
			continue
		}

		// Boundary Validation: First boarding must be origin, last dropping must be destination
		firstBoardingStop, err := findStop(tx, g.boarding[0].StopName, g.boarding[0].City)
		if err != nil || firstBoardingStop.ID != bus.OriginStopID {
			log.Printf("[seed] skipping %s (%s): first boarding stop (%s) does not match bus origin\n",
				key.BusNumber, key.TravelDate, g.boarding[0].StopName)
			continue
		}

		lastDroppingStop, err := findStop(tx, g.dropping[len(g.dropping)-1].StopName, g.dropping[len(g.dropping)-1].City)
		if err != nil || lastDroppingStop.ID != bus.DestinationStopID {
			log.Printf("[seed] skipping %s (%s): last dropping stop (%s) does not match bus destination\n",
				key.BusNumber, key.TravelDate, g.dropping[len(g.dropping)-1].StopName)
			continue
		}

		// 4. Insertion
		if isBoardingMode {
			for i, p := range g.boarding {
				stop, _ := findStop(tx, p.StopName, p.City)
				bp := model.BoardingPoint{
					BusInstanceID: inst.ID,
					BusStopID:     stop.ID,
					PickupTime:    p.Time,
					SequenceOrder: i + 1,
					Landmark:      p.Landmark,
				}
				tx.Where("bus_instance_id = ? AND bus_stop_id = ?", inst.ID, stop.ID).FirstOrCreate(&bp)
			}
		} else {
			startSeq := len(g.boarding) + 1
			for i, p := range g.dropping {
				stop, _ := findStop(tx, p.StopName, p.City)
				dp := model.DroppingPoint{
					BusInstanceID: inst.ID,
					BusStopID:     stop.ID,
					DropTime:      p.Time,
					SequenceOrder: startSeq + i,
					Landmark:      p.Landmark,
				}
				tx.Where("bus_instance_id = ? AND bus_stop_id = ?", inst.ID, stop.ID).FirstOrCreate(&dp)
			}
		}
	}
	return nil
}

func findStop(tx *gorm.DB, name, city string) (*model.BusStop, error) {
	var stop model.BusStop
	if city != "" {
		if err := tx.Where("name = ? AND city = ?", name, city).First(&stop).Error; err != nil {
			return nil, err
		}
	} else {
		if err := tx.Where("name = ?", name).First(&stop).Error; err != nil {
			return nil, err
		}
	}
	return &stop, nil
}

