package dto

type FareResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	SeatType        string  `json:"seat_type"`
	Price           float64 `json:"price"`
	Refundable      bool    `json:"refundable"`
	CancellationFee float64 `json:"cancellation_fee"`
	DateChangeFee   float64 `json:"date_change_fee"`
	SeatsAvailable  int     `json:"seats_available"`
}

type SeatResponse struct {
	ID          string  `json:"id"`
	SeatNumber  string  `json:"seat_number"`
	SeatType    string  `json:"seat_type"`
	BerthType   string  `json:"berth_type"`
	Position    string  `json:"position"`
	ExtraCharge float64 `json:"extra_charge"`
	IsAvailable bool    `json:"is_available"`
	Category    string  `json:"category"`
}
