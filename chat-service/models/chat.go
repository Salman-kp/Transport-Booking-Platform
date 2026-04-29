package models

import "time"

type ChatMessage struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string    `json:"user_id" gorm:"index;not null"`
	Sender    string    `json:"sender" gorm:"not null"` // "USER" or "ADMIN"
	Content   string    `json:"content" gorm:"not null"`
	IsDeleted bool      `json:"is_deleted" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}
