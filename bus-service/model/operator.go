package model

import (
	"time"

	"github.com/google/uuid"
)

type Operator struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name           string    `gorm:"type:varchar(200);not null" json:"name"`
	OperatorCode   string    `gorm:"type:varchar(10);not null;unique" json:"operator_code"`
	ContactEmail   string    `gorm:"type:varchar(200)" json:"contact_email"`
	ContactPhone   string    `gorm:"type:varchar(20)" json:"contact_phone"`
	LogoURL        string    `gorm:"type:text" json:"logo_url"`
	Rating         float64   `gorm:"type:decimal(3,2);default:0.00" json:"rating"`
	CommissionRate float64   `gorm:"type:decimal(5,2);not null;default:5.00" json:"commission_rate"`
	Status         string    `gorm:"type:varchar(20);not null;default:'PENDING'" json:"status"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time `gorm:"default:now()" json:"created_at"`
	UpdatedAt      time.Time `gorm:"default:now()" json:"updated_at"`
}

type OperatorUser struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;unique" json:"user_id"`
	OperatorID  uuid.UUID `gorm:"type:uuid;not null" json:"operator_id"`
	Role        string    `gorm:"type:varchar(20);not null;default:'MANAGER'" json:"role"`
	CreditLimit float64   `gorm:"type:decimal(12,2);not null;default:0" json:"credit_limit"`
	CreditUsed  float64   `gorm:"type:decimal(12,2);not null;default:0" json:"credit_used"`
	Status      string    `gorm:"type:varchar(20);not null;default:'PENDING'" json:"status"`
	CreatedAt   time.Time `gorm:"default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"default:now()" json:"updated_at"`

	Operator Operator `gorm:"foreignKey:OperatorID" json:"operator"`
}

type OperatorInventory struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OperatorID     uuid.UUID `gorm:"type:uuid;not null" json:"operator_id"`
	BusInstanceID  uuid.UUID `gorm:"type:uuid;not null" json:"bus_instance_id"`
	FareTypeID     uuid.UUID `gorm:"type:uuid;not null" json:"fare_type_id"`
	SeatType       string    `gorm:"type:varchar(20);not null" json:"seat_type"`
	QuantityLoaded int       `gorm:"not null" json:"quantity_loaded"`
	QuantitySold   int       `gorm:"not null;default:0" json:"quantity_sold"`
	WholesalePrice float64   `gorm:"type:decimal(10,2);not null" json:"wholesale_price"`
	SellingPrice   float64   `gorm:"type:decimal(10,2);not null" json:"selling_price"`
	Status         string    `gorm:"type:varchar(20);not null;default:'ACTIVE'" json:"status"`
	LoadedAt       time.Time `gorm:"default:now()" json:"loaded_at"`
	ExpiresAt      time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt      time.Time `gorm:"default:now()" json:"created_at"`
	UpdatedAt      time.Time `gorm:"default:now()" json:"updated_at"`

	Operator Operator `gorm:"foreignKey:OperatorID" json:"operator"`
}
