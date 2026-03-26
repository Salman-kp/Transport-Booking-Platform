package dto

import "time"

type FlightSearchRequest struct {
	Origin        string `query:"origin"`
	Destination   string `query:"destination"`
	DepartureDate string `query:"departure_date"` // Format YYYY-MM-DD
	ReturnDate    string `query:"return_date"`    // Optional
	Passengers    int    `query:"passengers"`
	Class         string `query:"class"`
	TripType      string `query:"trip_type"`
	Stops         *int   `query:"stops"`
	Airline       string `query:"airline"`
	MinPrice      *int   `query:"min_price"`
	MaxPrice      *int   `query:"max_price"`
	SortBy        string `query:"sort_by"`
}

// FareResponse represents individual pricing tiers matching the flight
type FareResponse struct {
	Class string  `json:"class"`
	Name  string  `json:"name"` // Saver, Flexi, Super Flexi
	Price float64 `json:"price"`
}

// FlightSearchResponse summarizes exactly what the web browser needs to render a flight card.
type FlightSearchResponse struct {
	InstanceID      string         `json:"instance_id"`
	FlightNumber    string         `json:"flight_number"`
	AirlineName     string         `json:"airline_name"`
	AirlineLogo     string         `json:"airline_logo"`
	Origin          string         `json:"origin"`
	Destination     string         `json:"destination"`
	DepartureTime   time.Time      `json:"departure_time"`
	ArrivalTime     time.Time      `json:"arrival_time"`
	DurationMinutes int            `json:"duration_minutes"`
	Fares           []FareResponse `json:"fares"`
}

type InstanceDetailsResponse struct {
	InstanceID      string    `json:"instance_id"`
	FlightNumber    string    `json:"flight_number"`
	AirlineName     string    `json:"airline_name"`
	Origin          string    `json:"origin"`
	Destination     string    `json:"destination"`
	DepartureTime   time.Time `json:"departure_time"`
	ArrivalTime     time.Time `json:"arrival_time"`
	DurationMinutes int       `json:"duration_minutes"`
	Status          string    `json:"status"`
}

type SeatDto struct {
	SeatNumber  string  `json:"seat_number"`
	SeatClass   string  `json:"seat_class"`
	IsAvailable bool    `json:"is_available"`
	ExtraCharge float64 `json:"extra_charge"`
}

type SeatMapResponse struct {
	EconomySeats  []SeatDto `json:"economy_seats"`
	BusinessSeats []SeatDto `json:"business_seats"`
}

type AncillaryDto struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type FarePredictionResponse struct {
	Signal     string `json:"signal"`
	Confidence int    `json:"confidence"`
	Reason     string `json:"reason"`
}
