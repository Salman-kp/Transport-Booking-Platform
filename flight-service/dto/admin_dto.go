package dto

type CreateFlightRequest struct {
	FlightNumber         string  `json:"flight_number"`
	AirlineID            string  `json:"airline_id"`
	AircraftTypeID       string  `json:"aircraft_type_id"`
	OriginAirportID      string  `json:"origin_airport_id"`
	DestinationAirportID string  `json:"destination_airport_id"`
	DepartureTime        string  `json:"departure_time"` // RFC3339
	ArrivalTime          string  `json:"arrival_time"`   // RFC3339
	DurationMinutes      int     `json:"duration_minutes"`
	DaysOfWeek           []int64 `json:"days_of_week"`
	IsActive             *bool   `json:"is_active"`
}
