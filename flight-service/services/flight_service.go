package services

import (
	"strings"
	"time"

	"github.com/junaid9001/tripneo/flight-service/domain_errors"
	"github.com/junaid9001/tripneo/flight-service/dto"
	"github.com/junaid9001/tripneo/flight-service/repository"
)

type FlightService struct {
	repo *repository.FlightRepository
}

func NewFlightService(repo *repository.FlightRepository) *FlightService {
	return &FlightService{repo: repo}
}

func (s *FlightService) SearchFlights(req dto.FlightSearchRequest) ([]dto.FlightSearchResponse, error) {
	searchDate, err := time.Parse("2006-01-02", req.DepartureDate)
	if err != nil {
		return nil, domain_errors.ErrInvalidDate
	}

	instances, err := s.repo.FindFlights(req, searchDate)
	if err != nil {
		return nil, domain_errors.ErrDatabaseQuery
	}

	if len(instances) == 0 {
		return []dto.FlightSearchResponse{}, nil
	}

	var instanceIDs []string
	for _, inst := range instances {
		instanceIDs = append(instanceIDs, inst.ID.String())
	}

	fares, err := s.repo.FindFareTypesForInstances(instanceIDs, strings.ToUpper(req.Class))
	if err != nil {
		return nil, domain_errors.ErrDatabaseQuery
	}

	var responses []dto.FlightSearchResponse
	for _, inst := range instances {
		flightDto := dto.FlightSearchResponse{
			InstanceID:      inst.ID.String(),
			FlightNumber:    inst.Flight.FlightNumber,
			AirlineName:     inst.Flight.Airline.Name,
			AirlineLogo:     inst.Flight.Airline.LogoUrl,
			Origin:          inst.Flight.OriginAirport.IataCode,
			Destination:     inst.Flight.DestinationAirport.IataCode,
			DepartureTime:   inst.DepartureAt,
			ArrivalTime:     inst.ArrivalAt,
			DurationMinutes: inst.Flight.DurationMinutes,
		}

		for _, f := range fares {
			if f.FlightInstanceID == inst.ID {
				flightDto.Fares = append(flightDto.Fares, dto.FareResponse{
					Class: f.SeatClass,
					Name:  f.Name,
					Price: f.Price,
				})
			}
		}

		responses = append(responses, flightDto)
	}

	return responses, nil
}

func (s *FlightService) GetFlightDetails(instanceId string) (*dto.InstanceDetailsResponse, error) {
	inst, err := s.repo.GetInstanceByID(instanceId)
	if err != nil {
		return nil, domain_errors.ErrFlightNotFound
	}
	return &dto.InstanceDetailsResponse{
		InstanceID:      inst.ID.String(),
		FlightNumber:    inst.Flight.FlightNumber,
		AirlineName:     inst.Flight.Airline.Name,
		Origin:          inst.Flight.OriginAirport.IataCode,
		Destination:     inst.Flight.DestinationAirport.IataCode,
		DepartureTime:   inst.DepartureAt,
		ArrivalTime:     inst.ArrivalAt,
		DurationMinutes: inst.Flight.DurationMinutes,
		Status:          string(inst.Status),
	}, nil
}

func (s *FlightService) GetFares(instanceId string) ([]dto.FareResponse, error) {
	fares, err := s.repo.GetFaresByInstanceID(instanceId)
	if err != nil {
		return nil, domain_errors.ErrDatabaseQuery
	}
	var responses []dto.FareResponse
	for _, f := range fares {
		responses = append(responses, dto.FareResponse{Class: f.SeatClass, Name: f.Name, Price: f.Price})
	}
	return responses, nil
}

func (s *FlightService) GetSeatMap(instanceId string) (*dto.SeatMapResponse, error) {
	seats, err := s.repo.GetSeatsByInstanceID(instanceId)
	if err != nil {
		return nil, domain_errors.ErrDatabaseQuery
	}
	res := &dto.SeatMapResponse{}
	for _, st := range seats {
		mapped := dto.SeatDto{SeatNumber: st.SeatNumber, SeatClass: st.SeatClass, IsAvailable: st.IsAvailable, ExtraCharge: st.ExtraCharge}
		if st.SeatClass == "ECONOMY" {
			res.EconomySeats = append(res.EconomySeats, mapped)
		} else {
			res.BusinessSeats = append(res.BusinessSeats, mapped)
		}
	}
	return res, nil
}

func (s *FlightService) GetAncillaries(instanceId string) ([]dto.AncillaryDto, error) {
	return []dto.AncillaryDto{
		{ID: "BAG_15", Type: "baggage", Description: "Extra 15kg Check-in", Price: 2500},
		{ID: "MEAL_VEG", Type: "meal", Description: "Premium Veg Meal", Price: 650},
	}, nil
}

func (s *FlightService) GetFarePrediction(instanceId string) (*dto.FarePredictionResponse, error) {
	return &dto.FarePredictionResponse{Signal: "book_now", Confidence: 92, Reason: "Historical AI trends indicate fares will aggressively spike due to the upcoming weekend volume."},
		nil
}
