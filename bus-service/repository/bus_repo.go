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
}

type busRepository struct {
	db *gorm.DB
}

func NewBusRepository(db *gorm.DB) BusRepository {
	return &busRepository{db: db}
}

func (r *busRepository) SearchBuses(filter model.SearchBusFilter) ([]model.BusInstance, error) {
	var instances []model.BusInstance
	
	query := r.db.Preload("Bus").Preload("Bus.Operator").Preload("Bus.BusType").
		Joins("JOIN buses ON buses.id = bus_instances.bus_id").
		Joins("JOIN bus_stops AS origin_stop ON origin_stop.id = buses.origin_stop_id").
		Joins("JOIN bus_stops AS dest_stop ON dest_stop.id = buses.destination_stop_id").
		Joins("JOIN operators ON operators.id = buses.operator_id").
		Where("DATE(bus_instances.travel_date) = ?", filter.TravelDate)

	if filter.Origin != "" {
		query = query.Where("origin_stop.name ILIKE ?", "%"+filter.Origin+"%")
	}
	if filter.Destination != "" {
		query = query.Where("dest_stop.name ILIKE ?", "%"+filter.Destination+"%")
	}

	// Dynamic capacity checks based on SeatType request
	if filter.SeatType != "" && filter.Passengers > 0 {
		switch filter.SeatType {
		case "seater":
			query = query.Where("bus_instances.available_seater >= ?", filter.Passengers)
		case "semi_sleeper":
			query = query.Where("bus_instances.available_semi_sleeper >= ?", filter.Passengers)
		case "sleeper":
			query = query.Where("bus_instances.available_sleeper >= ?", filter.Passengers)
		}
	} else if filter.Passengers > 0 {
		query = query.Where("(bus_instances.available_seater >= ? OR bus_instances.available_semi_sleeper >= ? OR bus_instances.available_sleeper >= ?)", filter.Passengers, filter.Passengers, filter.Passengers)
	}

	// Operator Filter
	if filter.Operator != "" {
		query = query.Where("operators.operator_code = ?", filter.Operator)
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
			query = query.Where("buses.departure_time >= '06:00:00' AND buses.departure_time < '12:00:00'")
		case "afternoon":
			query = query.Where("buses.departure_time >= '12:00:00' AND buses.departure_time < '17:00:00'")
		case "evening":
			query = query.Where("buses.departure_time >= '17:00:00' AND buses.departure_time < '21:00:00'")
		case "night":
			query = query.Where("buses.departure_time >= '21:00:00' OR buses.departure_time < '06:00:00'")
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
		query = query.Order("buses.departure_time ASC")
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
