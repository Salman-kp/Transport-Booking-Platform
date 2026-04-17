package repository

import (
	"github.com/Salman-kp/tripneo/bus-service/model"
	"gorm.io/gorm"
)

type BusRepository interface {
	SearchBuses(filter model.SearchBusFilter) ([]model.BusInstance, error)
	GetBusInstanceByID(id string) (*model.BusInstance, error)
	GetFaresByInstanceID(id string) ([]model.FareType, error)
	GetSeatsByInstanceID(id string) ([]model.Seat, error)
	GetAmenitiesByInstanceID(id string) (interface{}, error)
	GetBoardingPointsByInstanceID(id string) ([]model.BoardingPoint, error)
	GetDroppingPointsByInstanceID(id string) ([]model.DroppingPoint, error)
	SearchBusStops(search string) ([]model.BusStop, error)
	GetAllOperators() ([]model.Operator, error)
}

type busRepository struct {
	db *gorm.DB
}

func NewBusRepository(db *gorm.DB) BusRepository {
	return &busRepository{db: db}
}

func (r *busRepository) SearchBuses(filter model.SearchBusFilter) ([]model.BusInstance, error) {
	instances := make([]model.BusInstance, 0)

	query := r.db.Preload("Bus").Preload("Bus.Operator").Preload("Bus.BusType").
		Preload("Bus.OriginStop").Preload("Bus.DestinationStop").
		Distinct("bus_instances.*").
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Joins("JOIN operators ON operators.id = buses.operator_id").
		// Join boarding side: user's origin stop
		Joins("JOIN boarding_points bp ON bp.bus_instance_id = bus_instances.id").
		Joins("JOIN bus_stops bs_from ON bs_from.id = bp.bus_stop_id").
		// Join dropping side: user's destination stop
		Joins("JOIN dropping_points dp ON dp.bus_instance_id = bus_instances.id").
		Joins("JOIN bus_stops bs_to ON bs_to.id = dp.bus_stop_id").
		// Enforce forward direction: boarding must come before dropping in route sequence
		Where("bp.sequence_order < dp.sequence_order").
		Where("DATE(bus_instances.travel_date) = ?", filter.TravelDate)

	if filter.Origin != "" {
		query = query.Where(
			"(bs_from.name ILIKE ? OR bs_from.city ILIKE ?)",
			"%"+filter.Origin+"%", "%"+filter.Origin+"%",
		)
	}
	if filter.Destination != "" {
		query = query.Where(
			"(bs_to.name ILIKE ? OR bs_to.city ILIKE ?)",
			"%"+filter.Destination+"%", "%"+filter.Destination+"%",
		)
	}

	// Seat capacity filters
	if filter.SeatType != "" && filter.Passengers > 0 {
		switch filter.SeatType {
		case "seater":
			query = query.Where("bus_instances.available_seater >= ?", filter.Passengers)
		case "semi_sleeper", "semi-sleeper":
			query = query.Where("bus_instances.available_semi_sleeper >= ?", filter.Passengers)
		case "sleeper":
			query = query.Where("bus_instances.available_sleeper >= ?", filter.Passengers)
		}
	} else if filter.Passengers > 0 {
		query = query.Where(
			"(bus_instances.available_seater >= ? OR bus_instances.available_semi_sleeper >= ? OR bus_instances.available_sleeper >= ?)",
			filter.Passengers, filter.Passengers, filter.Passengers,
		)
	}

	// Operator Filter
	if filter.Operator != "" {
		query = query.Where("operators.operator_code ILIKE ?", "%"+filter.Operator+"%")
	}

	// Price range check
	if filter.MinPrice > 0 {
		if filter.SeatType == "sleeper" {
			query = query.Where("bus_instances.current_price_sleeper >= ?", filter.MinPrice)
		} else {
			query = query.Where("bus_instances.current_price_seater >= ?", filter.MinPrice)
		}
	}
	if filter.MaxPrice > 0 {
		if filter.SeatType == "sleeper" {
			query = query.Where("bus_instances.current_price_sleeper <= ?", filter.MaxPrice)
		} else {
			query = query.Where("bus_instances.current_price_seater <= ?", filter.MaxPrice)
		}
	}

	// Departure block filtering
	if filter.DepartureTime != "" {
		switch filter.DepartureTime {
		case "morning":
			query = query.Where("buses.departure_time::time >= '06:00:00'::time AND buses.departure_time::time < '12:00:00'::time")
		case "afternoon":
			query = query.Where("buses.departure_time::time >= '12:00:00'::time AND buses.departure_time::time < '17:00:00'::time")
		case "evening":
			query = query.Where("buses.departure_time::time >= '17:00:00'::time AND buses.departure_time::time < '21:00:00'::time")
		case "night":
			query = query.Where("buses.departure_time::time >= '21:00:00'::time OR buses.departure_time::time < '06:00:00'::time")
		}
	}

	// Unified Sorting Logic
	switch filter.SortBy {
	case "price":
		if filter.SeatType == "sleeper" {
			query = query.Order("bus_instances.current_price_sleeper ASC")
		} else if filter.SeatType == "semi_sleeper" {
			query = query.Order("bus_instances.current_price_semi_sleeper ASC")
		} else {
			query = query.Order("bus_instances.current_price_seater ASC")
		}
	case "duration":
		query = query.Order("buses.duration_minutes ASC")
	case "departure_time":
		query = query.Order("bus_instances.departure_at ASC")
	case "rating":
		query = query.Order("operators.rating DESC")
	}

	err := query.Find(&instances).Error
	return instances, err
}

func (r *busRepository) GetBusInstanceByID(id string) (*model.BusInstance, error) {
	var instance model.BusInstance
	err := r.db.Preload("Bus").Preload("Bus.Operator").Preload("Bus.BusType").
		Preload("Bus.OriginStop").Preload("Bus.DestinationStop").
		First(&instance, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (r *busRepository) GetFaresByInstanceID(id string) ([]model.FareType, error) {
	var fares []model.FareType
	err := r.db.Where("bus_instance_id = ?", id).Find(&fares).Error
	return fares, err
}

func (r *busRepository) GetSeatsByInstanceID(id string) ([]model.Seat, error) {
	var seats []model.Seat
	err := r.db.Where("bus_instance_id = ?", id).Find(&seats).Error
	return seats, err
}

func (r *busRepository) GetAmenitiesByInstanceID(id string) (interface{}, error) {
	var instance model.BusInstance
	err := r.db.Preload("Bus.BusType").First(&instance, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return instance.Bus.BusType.Amenities, nil
}

func (r *busRepository) GetBoardingPointsByInstanceID(id string) ([]model.BoardingPoint, error) {
	var points []model.BoardingPoint
	err := r.db.Preload("BusStop").Where("bus_instance_id = ?", id).Order("sequence_order ASC").Find(&points).Error
	return points, err
}

func (r *busRepository) GetDroppingPointsByInstanceID(id string) ([]model.DroppingPoint, error) {
	var points []model.DroppingPoint
	err := r.db.Preload("BusStop").Where("bus_instance_id = ?", id).Order("sequence_order ASC").Find(&points).Error
	return points, err
}

func (r *busRepository) SearchBusStops(search string) ([]model.BusStop, error) {
	var stops []model.BusStop
	query := r.db.Order("name ASC")
	if search != "" {
		query = query.Where("name ILIKE ? OR city ILIKE ? OR state ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	err := query.Find(&stops).Error
	return stops, err
}

func (r *busRepository) GetAllOperators() ([]model.Operator, error) {
	var operators []model.Operator
	err := r.db.Where("is_active = true").Order("name ASC").Find(&operators).Error
	return operators, err
}
