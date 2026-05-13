package model

import (
	"time"

	"github.com/google/uuid"
)

type PrebookingAccounting struct {
	ID                  uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InstanceID          uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"instance_id"`
	OperatorName        string    `gorm:"type:varchar(200);not null" json:"operator_name"`
	BusNumber           string    `gorm:"type:varchar(50);not null" json:"bus_number"`
	DepartureDateTime   time.Time `gorm:"not null" json:"departure_date_time"`
	ArrivalDateTime     time.Time `gorm:"not null" json:"arrival_date_time"`
	TotalPurchasedSeats int       `gorm:"not null" json:"total_purchased_seats"`
	OriginalSeatPrice   float64   `gorm:"type:decimal(10,2);not null" json:"original_seat_price"`
	SellingSeatPrice    float64   `gorm:"type:decimal(10,2);not null" json:"selling_seat_price"`
	SpendAmountTotal    float64   `gorm:"type:decimal(10,2);not null" json:"spend_amount_total"`
	ProfitAmount        float64   `gorm:"type:decimal(10,2);default:0" json:"profit_amount"`
	LossAmount          float64   `gorm:"type:decimal(10,2);default:0" json:"loss_amount"`
	IsFinalized         bool      `gorm:"default:false" json:"is_finalized"`
	CreatedAt           time.Time `gorm:"default:now()" json:"created_at"`
	UpdatedAt           time.Time `gorm:"default:now()" json:"updated_at"`

	BusInstance BusInstance `gorm:"foreignKey:InstanceID" json:"bus_instance"`
}
