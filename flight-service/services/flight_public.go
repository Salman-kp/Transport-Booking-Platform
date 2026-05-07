package services

import (
	"github.com/junaid9001/tripneo/flight-service/models"
)

func (s *FlightService) SearchAirport(search string) ([]models.Airport, error) {
	airports, err := s.repo.SearchAirport(search)
	if err != nil {
		return airports, err
	}
	return airports, nil
}

func (s *FlightService) GetAirlines() ([]models.Airline, error) {
	airlines, err := s.repo.GetAirlines()
	if err != nil {
		return nil, err
	}

	return airlines, nil
}

func (s *FlightService) GetAircraftTypes() ([]models.AircraftType, error) {
	return s.repo.GetAircraftTypes()
}
