package models

type EstadoSolicitudItem struct {
	Codigo      string `gorm:"primaryKey;size:20"`
	Nombre      string `gorm:"size:50;not null"`
	Color       string `gorm:"size:20;default:'gray'"`
	Descripcion string `gorm:"size:255"`
}

func (EstadoSolicitudItem) TableName() string {
	return "estados_solicitud_item"
}
