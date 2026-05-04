package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name         string    `gorm:"size:20;not null"`
	Email        string    `gorm:"size:254;uniqueIndex;not null"`
	PasswordHash string
	Role         string   `gorm:"default:'user'"`
	Permissions  []string `gorm:"serializer:json"`
	IsVerified   bool     `gorm:"default:false"`
	CreatedAt    time.Time
}
