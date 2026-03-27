package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type PricingRule struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name       string         `gorm:"type:varchar(100);not null" json:"name"`
	RuleType   string         `gorm:"type:varchar(30);not null" json:"rule_type"`
	Conditions datatypes.JSON `gorm:"type:jsonb;not null" json:"conditions"`
	Multiplier float64        `gorm:"type:decimal(5,3);not null" json:"multiplier"`
	Priority   int            `gorm:"not null;default:0" json:"priority"`
	IsActive   bool           `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time      `gorm:"default:now()" json:"created_at"`
}
