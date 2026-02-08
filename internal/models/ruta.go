package models

type Ruta struct {
	BaseModel
	Tramo     string `gorm:"size:255;not null;unique"`
	Sigla     string `gorm:"size:50"`
	NacInter  string `gorm:"size:50;not null"`
	Ubicacion string `gorm:"size:100"`
	Medio     string `gorm:"size:50;default:'AEREO'"`

	OrigenIATA string  `gorm:"size:5;not null"`
	Origen     Destino `gorm:"foreignKey:OrigenIATA;references:IATA"`

	DestinoIATA string  `gorm:"size:5;not null"`
	Destino     Destino `gorm:"foreignKey:DestinoIATA;references:IATA"`

	Escalas   []RutaEscala   `gorm:"foreignKey:RutaID"`
	Contratos []RutaContrato `gorm:"foreignKey:RutaID"`
}

type RutaEscala struct {
	BaseModel
	RutaID      string  `gorm:"size:36;not null;index"`
	DestinoIATA string  `gorm:"size:5;not null"`
	Destino     Destino `gorm:"foreignKey:DestinoIATA;references:IATA"`
	Orden       int     `gorm:"not null"`
}

func (Ruta) TableName() string       { return "rutas" }
func (RutaEscala) TableName() string { return "ruta_escalas" }
