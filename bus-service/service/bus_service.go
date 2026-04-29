package service

import (
	"sort"

	"github.com/Salman-kp/tripneo/bus-service/dto"
	"github.com/Salman-kp/tripneo/bus-service/model"
	"github.com/Salman-kp/tripneo/bus-service/repository"
)

type BusService interface {
	SearchBuses(filter model.SearchBusFilter) ([]model.BusInstance, error)
	GetBusInstance(id string) (*model.BusInstance, error)
	GetFares(id string) ([]dto.FareResponse, error)
	GetSeats(id string) ([]dto.SeatResponse, error)
	GetAmenities(id string) (interface{}, error)
	GetBoardingPoints(id string) ([]dto.BoardingPointResponse, error)
	GetDroppingPoints(id string) ([]dto.DroppingPointResponse, error)
	GetRoute(id string) ([]dto.RouteStop, error)
	GetBusStops(search string) ([]model.BusStop, error)
	GetOperators() ([]model.Operator, error)
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

func (s *busService) GetFares(id string) ([]dto.FareResponse, error) {
	fares, err := s.repo.GetFaresByInstanceID(id)
	if err != nil {
		return nil, err
	}

	res := make([]dto.FareResponse, 0, len(fares))

	for _, f := range fares {
		res = append(res, dto.FareResponse{
			ID:              f.ID.String(),
			Name:            f.Name,
			SeatType:        f.SeatType,
			Price:           f.Price,
			Refundable:      f.IsRefundable,
			CancellationFee: f.CancellationFee,
			DateChangeFee:   f.DateChangeFee,
			SeatsAvailable:  f.SeatsAvailable,
		})
	}

	return res, nil
}

func (s *busService) GetSeats(id string) ([]dto.SeatResponse, error) {
	seats, err := s.repo.GetSeatsByInstanceID(id)
	if err != nil {
		return nil, err
	}

	res := make([]dto.SeatResponse, 0, len(seats))

	for _, seat := range seats {
		res = append(res, dto.SeatResponse{
			ID:          seat.ID.String(),
			SeatNumber:  seat.SeatNumber,
			SeatType:    seat.SeatType,
			BerthType:   seat.BerthType,
			Position:    seat.Position,
			ExtraCharge: seat.ExtraCharge,
			IsAvailable: seat.IsAvailable,
			Category:    seat.Category,
		})
	}

	return res, nil
}

func (s *busService) GetAmenities(id string) (interface{}, error) {
	return s.repo.GetAmenitiesByInstanceID(id)
}

func (s *busService) GetBoardingPoints(id string) ([]dto.BoardingPointResponse, error) {
	boarding, err := s.repo.GetBoardingPointsByInstanceID(id)
	if err != nil {
		return nil, err
	}

	res := make([]dto.BoardingPointResponse, 0, len(boarding))
	for _, b := range boarding {
		res = append(res, dto.BoardingPointResponse{
			ID:            b.ID.String(),
			StopID:        b.BusStopID.String(),
			StopName:      b.BusStop.Name,
			City:          b.BusStop.City,
			Latitude:      b.BusStop.Latitude,
			Longitude:     b.BusStop.Longitude,
			PickupTime:    b.PickupTime,
			SequenceOrder: b.SequenceOrder,
			Landmark:      b.Landmark,
		})
	}

	return res, nil
}

func (s *busService) GetDroppingPoints(id string) ([]dto.DroppingPointResponse, error) {
	dropping, err := s.repo.GetDroppingPointsByInstanceID(id)
	if err != nil {
		return nil, err
	}

	res := make([]dto.DroppingPointResponse, 0, len(dropping))
	for _, d := range dropping {
		res = append(res, dto.DroppingPointResponse{
			ID:            d.ID.String(),
			StopID:        d.BusStopID.String(),
			StopName:      d.BusStop.Name,
			City:          d.BusStop.City,
			Latitude:      d.BusStop.Latitude,
			Longitude:     d.BusStop.Longitude,
			DropTime:      d.DropTime,
			SequenceOrder: d.SequenceOrder,
			Landmark:      d.Landmark,
		})
	}

	return res, nil
}

func (s *busService) GetRoute(id string) ([]dto.RouteStop, error) {
	boarding, err := s.repo.GetBoardingPointsByInstanceID(id)
	if err != nil {
		return nil, err
	}
	dropping, err := s.repo.GetDroppingPointsByInstanceID(id)
	if err != nil {
		return nil, err
	}

	route := make([]dto.RouteStop, 0, len(boarding)+len(dropping))

	for _, b := range boarding {
		route = append(route, dto.RouteStop{
			StopName:  b.BusStop.Name,
			City:      b.BusStop.City,
			Latitude:  b.BusStop.Latitude,
			Longitude: b.BusStop.Longitude,
			Time:      b.PickupTime,
			Type:      "boarding",
			Sequence:  b.SequenceOrder,
		})
	}

	for _, d := range dropping {
		route = append(route, dto.RouteStop{
			StopName:  d.BusStop.Name,
			City:      d.BusStop.City,
			Latitude:  d.BusStop.Latitude,
			Longitude: d.BusStop.Longitude,
			Time:      d.DropTime,
			Type:      "dropping",
			Sequence:  d.SequenceOrder,
		})
	}

	sort.Slice(route, func(i, j int) bool {
		return route[i].Sequence < route[j].Sequence
	})

	return route, nil
}

func (s *busService) GetBusStops(search string) ([]model.BusStop, error) {
	return s.repo.SearchBusStops(search)
}

func (s *busService) GetOperators() ([]model.Operator, error) {
	return s.repo.GetAllOperators()
}
