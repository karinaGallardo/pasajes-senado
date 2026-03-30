package models

import (
	"time"
)

type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Action     string    `gorm:"size:100;not null;index" json:"action"`     // e.g., "APROBAR_SOLICITUD", "EMITIR_PASAJE"
	EntityType string    `gorm:"size:50;not null;index" json:"entity_type"` // e.g., "solicitud", "pasaje", "descargo"
	EntityID   string    `gorm:"size:50;not null;index" json:"entity_id"`
	OldValue   string    `gorm:"type:text" json:"old_value"` // JSON or plain status
	NewValue   string    `gorm:"type:text" json:"new_value"`
	UserID     *string   `gorm:"size:50;index" json:"user_id"`
	Usuario    *Usuario  `gorm:"foreignKey:UserID" json:"usuario"`
	IP         string    `gorm:"size:45" json:"ip"`
	UserAgent  string    `gorm:"type:text" json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}
