package repository

import (
	"strings"
	"time"

	"github.com/junaid9001/tripneo/flight-service/dto"
	"github.com/junaid9001/tripneo/flight-service/models"
	"gorm.io/gorm"
)

type FlightRepository struct {
	db *gorm.DB
}

func NewFlightRepository(db *gorm.DB) *FlightRepository {
	return &FlightRepository{db: db}
}

func (r *FlightRepository) FindFlights(req dto.FlightSearchRequest, searchDate time.Time) ([]models.FlightInstance, error) {
	var instances []models.FlightInstance

	query := r.db.Model(&models.FlightInstance{}).
		Joins("JOIN flights ON flights.id = flight_instances.flight_id").
		Joins("JOIN airports origin_port ON origin_port.id = flights.origin_airport_id").
		Joins("JOIN airports dest_port ON dest_port.id = flights.destination_airport_id").
		Where("origin_port.iata_code = ?", req.Origin).
		Where("dest_port.iata_code = ?", req.Destination).
		Where("DATE(flight_instances.flight_date) = DATE(?)", searchDate).
		Where("flight_instances.status != ?", models.CANCELLED)

	query = query.Preload("Flight").
		Preload("Flight.Airline").
		Preload("Flight.OriginAirport").
		Preload("Flight.DestinationAirport")

	className := strings.ToUpper(req.Class)
	if className == "BUSINESS" {
		query = query.Where("flight_instances.available_business >= ?", req.Passengers)
	} else {
		query = query.Where("flight_instances.available_economy >= ?", req.Passengers)
	}

	if err := query.Find(&instances).Error; err != nil {
		return nil, err
	}
	return instances, nil
}

func (r *FlightRepository) FindFareTypesForInstances(instanceIDs []string, className string) ([]models.FareType, error) {
	var fares []models.FareType
	if len(instanceIDs) == 0 {
		return fares, nil
	}

	err := r.db.Where("flight_instance_id IN ? AND seat_class = ?", instanceIDs, className).Find(&fares).Error
	return fares, err
}

func (r *FlightRepository) GetInstanceByID(id string) (*models.FlightInstance, error) {
	var instance models.FlightInstance
	err := r.db.Preload("Flight").Preload("Flight.Airline").Preload("Flight.OriginAirport").Preload("Flight.DestinationAirport").Where("id = ?", id).First(&instance).Error
	return &instance, err
}

func (r *FlightRepository) GetFaresByInstanceID(id string) ([]models.FareType, error) {
	var fares []models.FareType
	err := r.db.Where("flight_instance_id = ?", id).Find(&fares).Error
	return fares, err
}

func (r *FlightRepository) GetSeatsByInstanceID(id string) ([]models.Seat, error) {
	var seats []models.Seat
	err := r.db.Where("flight_instance_id = ?", id).Order("seat_number ASC").Find(&seats).Error
	return seats, err
}
