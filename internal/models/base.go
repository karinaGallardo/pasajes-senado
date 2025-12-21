package models

import (
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	ID        string    `gorm:"primaryKey;type:varchar(36);default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	CreatedBy *string `gorm:"size:36;default:null"`
	UpdatedBy *string `gorm:"size:36;default:null"`
	DeletedBy *string `gorm:"size:36;default:null"`
}
