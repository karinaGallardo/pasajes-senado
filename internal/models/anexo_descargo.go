package models

type AnexoDescargo struct {
	BaseModel
	DescargoOficialID string `gorm:"size:36;not null;index"`
	Archivo           string `gorm:"size:255;not null"`

	// Seq is an auto-incrementing field managed by DB to ensure atomic sequential ordering
	Seq int64 `gorm:"autoIncrement;not null;<-:false"`
}

func (AnexoDescargo) TableName() string {
	return "anexos_descargo"
}
