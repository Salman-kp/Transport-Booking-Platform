package dto

type RegisterOperatorReq struct {
	UserID         string  `json:"user_id"`
	Name           string  `json:"name"`
	OperatorCode   string  `json:"operator_code"`
	ContactEmail   string  `json:"contact_email"`
	ContactPhone   string  `json:"contact_phone"`
	LogoURL        string  `json:"logo_url"`
	CommissionRate float64 `json:"commission_rate"`
}

type LoadInventoryReq struct {
	BusInstanceID  string  `json:"bus_instance_id"`
	FareTypeID     string  `json:"fare_type_id"`
	SeatType       string  `json:"seat_type"`
	QuantityLoaded int     `json:"quantity_loaded"`
	WholesalePrice float64 `json:"wholesale_price"`
	SellingPrice   float64 `json:"selling_price"`
	ExpiresAt      string  `json:"expires_at"` // ISO8601 string
}
