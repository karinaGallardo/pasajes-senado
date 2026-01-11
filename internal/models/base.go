package models

import (
	"time"

	"gorm.io/gorm"
)

type BaseModel struct {
	ID        string         `gorm:"primaryKey;type:varchar(36);default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `gorm:"index;type:timestamp"`
	UpdatedAt time.Time      `gorm:"type:timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"index;type:timestamp"`

	CreatedBy *string `gorm:"size:36;default:null"`
	UpdatedBy *string `gorm:"size:36;default:null"`
	DeletedBy *string `gorm:"size:36;default:null"`
}
