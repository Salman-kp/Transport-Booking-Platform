package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type BusStop struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name      string    `gorm:"type:varchar(200);not null" json:"name"`
	City      string    `gorm:"type:varchar(100);not null" json:"city"`
	State     string    `gorm:"type:varchar(100);not null" json:"state"`
	Country   string    `gorm:"type:varchar(100);not null;default:'India'" json:"country"`
	Latitude  float64   `gorm:"type:decimal(10,7)" json:"latitude"`
	Longitude float64   `gorm:"type:decimal(10,7)" json:"longitude"`
	CreatedAt time.Time `gorm:"default:now()" json:"created_at"`
}

type BusType struct {
	ID           uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name         string         `gorm:"type:varchar(100);not null" json:"name"`
	Manufacturer string         `gorm:"type:varchar(100)" json:"manufacturer"`
	AC           bool           `gorm:"not null;default:true" json:"ac"`
	SeatLayout   datatypes.JSON `gorm:"type:jsonb;not null" json:"seat_layout"`
	Amenities    datatypes.JSON `gorm:"type:jsonb" json:"amenities"`
	CreatedAt    time.Time      `gorm:"default:now()" json:"created_at"`
}

type Bus struct {
	ID                uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BusNumber         string         `gorm:"type:varchar(20);not null" json:"bus_number"`
	OperatorID        uuid.UUID      `gorm:"type:uuid;not null" json:"operator_id"`
	BusTypeID         uuid.UUID      `gorm:"type:uuid;not null" json:"bus_type_id"`
	OriginStopID      uuid.UUID      `gorm:"type:uuid;not null" json:"origin_stop_id"`
	DestinationStopID uuid.UUID      `gorm:"type:uuid;not null" json:"destination_stop_id"`
	DepartureTime     string         `gorm:"type:time;not null" json:"departure_time"`
	ArrivalTime       string         `gorm:"type:time;not null" json:"arrival_time"`
	DurationMinutes   int            `gorm:"not null" json:"duration_minutes"`
	DaysOfWeek        datatypes.JSON `gorm:"type:jsonb;not null" json:"days_of_week"`
	DistanceKM        int            `json:"distance_km"`
	IsActive          bool           `gorm:"default:true" json:"is_active"`
	CreatedAt         time.Time      `gorm:"default:now()" json:"created_at"`

	Operator        Operator `gorm:"foreignKey:OperatorID" json:"operator"`
	BusType         BusType  `gorm:"foreignKey:BusTypeID" json:"bus_type"`
	OriginStop      BusStop  `gorm:"foreignKey:OriginStopID" json:"origin_stop"`
	DestinationStop BusStop  `gorm:"foreignKey:DestinationStopID" json:"destination_stop"`
}

type BusInstance struct {
	ID                      uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BusID                   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_bus_date" json:"bus_id"`
	DeviceID                string    `gorm:"type:varchar(100)" json:"device_id"`
	TravelDate              time.Time `gorm:"type:date;not null;uniqueIndex:idx_bus_date" json:"travel_date"`
	DepartureAt             time.Time `gorm:"not null" json:"departure_at"`
	ArrivalAt               time.Time `gorm:"not null" json:"arrival_at"`
	Status                  string    `gorm:"type:varchar(20);not null;default:'SCHEDULED'" json:"status"`
	DelayMinutes            int       `gorm:"default:0" json:"delay_minutes"`
	AvailableSeater         int       `gorm:"not null;default:0" json:"available_seater"`
	AvailableSemiSleeper    int       `gorm:"not null;default:0" json:"available_semi_sleeper"`
	AvailableSleeper        int       `gorm:"not null;default:0" json:"available_sleeper"`
	BasePriceSeater         float64   `gorm:"type:decimal(10,2);not null;default:0" json:"base_price_seater"`
	BasePriceSemiSleeper    float64   `gorm:"type:decimal(10,2);not null;default:0" json:"base_price_semi_sleeper"`
	BasePriceSleeper        float64   `gorm:"type:decimal(10,2);not null;default:0" json:"base_price_sleeper"`
	CurrentPriceSeater      float64   `gorm:"type:decimal(10,2);not null;default:0" json:"current_price_seater"`
	CurrentPriceSemiSleeper float64   `gorm:"type:decimal(10,2);not null;default:0" json:"current_price_semi_sleeper"`
	CurrentPriceSleeper     float64   `gorm:"type:decimal(10,2);not null;default:0" json:"current_price_sleeper"`
	CreatedAt               time.Time `gorm:"default:now()" json:"created_at"`
	UpdatedAt               time.Time `gorm:"default:now()" json:"updated_at"`

	Bus Bus `gorm:"foreignKey:BusID" json:"bus"`
}

type SearchBusFilter struct {
	Origin        string
	Destination   string
	TravelDate    string
	Passengers    int
	SeatType      string
	Operator      string
	MinPrice      float64
	MaxPrice      float64
	SortBy        string
	DepartureTime string
}

type BoardingPoint struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BusInstanceID uuid.UUID `gorm:"type:uuid;not null" json:"bus_instance_id"`
	BusStopID     uuid.UUID `gorm:"type:uuid;not null" json:"bus_stop_id"`
	PickupTime    time.Time `gorm:"not null" json:"pickup_time"`
	Landmark      string    `gorm:"type:varchar(200)" json:"landmark"`
	SequenceOrder int       `gorm:"not null" json:"sequence_order"`
	CreatedAt     time.Time `gorm:"default:now()" json:"created_at"`

	BusInstance BusInstance `gorm:"foreignKey:BusInstanceID" json:"bus_instance"`
	BusStop     BusStop     `gorm:"foreignKey:BusStopID" json:"bus_stop"`
}

type DroppingPoint struct {
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BusInstanceID uuid.UUID `gorm:"type:uuid;not null" json:"bus_instance_id"`
	BusStopID     uuid.UUID `gorm:"type:uuid;not null" json:"bus_stop_id"`
	DropTime      time.Time `gorm:"not null" json:"drop_time"`
	Landmark      string    `gorm:"type:varchar(200)" json:"landmark"`
	SequenceOrder int       `gorm:"not null" json:"sequence_order"`
	CreatedAt     time.Time `gorm:"default:now()" json:"created_at"`

	BusInstance BusInstance `gorm:"foreignKey:BusInstanceID" json:"bus_instance"`
	BusStop     BusStop     `gorm:"foreignKey:BusStopID" json:"bus_stop"`
}