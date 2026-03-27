package service

import (
	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/Salman-kp/tripneo/bus-service/repository"
)

type BusService interface {
	SearchBuses(filter model.SearchBusFilter) ([]model.BusInstance, error)
	GetBusInstance(id string) (*model.BusInstance, error)
	GetFares(id string) ([]model.FareType, error)
	GetSeats(id string) ([]model.Seat, error)
	GetAmenities(id string) (interface{}, error)
	GetBoardingPoints(id string) (interface{}, error)
}

type busService struct {
	repo repository.BusRepository
}

func NewBusService(repo repository.BusRepository) BusService {
	return &busService{repo: repo}
}

func (s *busService) SearchBuses(filter model.SearchBusFilter) ([]model.BusInstance, error) {
	return s.repo.SearchBuses(filter)
}

func (s *busService) GetBusInstance(id string) (*model.BusInstance, error) {
	return s.repo.GetBusInstanceByID(id)
}

func (s *busService) GetFares(id string) ([]model.FareType, error) {
	return s.repo.GetFaresByInstanceID(id)
}

func (s *busService) GetSeats(id string) ([]model.Seat, error) {
	return s.repo.GetSeatsByInstanceID(id)
}

func (s *busService) GetAmenities(id string) (interface{}, error) {
	return s.repo.GetAmenitiesByInstanceID(id)
}

func (s *busService) GetBoardingPoints(id string) (interface{}, error) {
	boarding, err := s.repo.GetBoardingPointsByInstanceID(id)
	if err != nil {
		return nil, err
	}
	dropping, err := s.repo.GetDroppingPointsByInstanceID(id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"boarding_points": boarding,
		"dropping_points": dropping,
	}, nil
}
