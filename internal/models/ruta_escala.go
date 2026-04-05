package models

type RutaEscala struct {
	BaseModel
	RutaID      string  `gorm:"size:36;not null;index"`
	DestinoIATA string  `gorm:"size:5;not null"`
	Destino     Destino `gorm:"foreignKey:DestinoIATA;references:IATA"`
	// Seq is an auto-incrementing field managed by DB to ensure atomic sequential ordering
	Seq int64 `gorm:"autoIncrement;not null;<-:false"`
}

func (RutaEscala) TableName() string { return "ruta_escalas" }
