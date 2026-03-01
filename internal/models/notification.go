package models

import "time"

type Notification struct {
	BaseModel
	UserID    string     `gorm:"size:36;index" json:"user_id"`
	Title     string     `gorm:"size:255" json:"title"`
	Message   string     `gorm:"type:text" json:"message"`
	Type      string     `gorm:"size:50" json:"type"` // e.g. "new_solicitud"
	TargetURL string     `gorm:"size:255" json:"target_url"`
	IsRead    bool       `gorm:"default:false" json:"is_read"`
	ReadAt    *time.Time `json:"read_at"`

	User *Usuario `gorm:"foreignKey:UserID" json:"-"`
}
