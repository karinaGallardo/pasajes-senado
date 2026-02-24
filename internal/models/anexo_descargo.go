package models

type AnexoDescargo struct {
	BaseModel
	DescargoOficialID string `gorm:"size:36;not null;index"`
	Archivo           string `gorm:"size:255;not null"`
	Orden             int    `gorm:"default:0"`
}

func (AnexoDescargo) TableName() string {
	return "anexos_descargo"
}
