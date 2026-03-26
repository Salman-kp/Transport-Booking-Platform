package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type AircraftType struct {
	ID           uuid.UUID      `gorm:"type:uuid;primarykey;default:gen_random_uuid()"`
	Model        string         `gorm:"not null;size:30"`
	Manufacturer string         `gorm:"not null;size:30"`
	SeatLayout   datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt    time.Time
}

//seat layout example

//  economy:{
// 	rows:number
// 	columns:A,B,C,"" ,D  -empty string = aisle space (walking space)
//  }
