package models

import "strings"

type Ruta struct {
	BaseModel
	Tramo     string `gorm:"size:255;not null;unique"`
	Sigla     string `gorm:"size:50;unique"`
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

func (r Ruta) GetRutaDisplay() string {
	if r.Origen.Ciudad != "" && r.Destino.Ciudad != "" {
		parts := []string{r.Origen.Ciudad + " (" + r.Origen.IATA + ")"}
		for _, e := range r.Escalas {
			if e.Destino.Ciudad != "" {
				parts = append(parts, e.Destino.Ciudad+" ("+e.Destino.IATA+")")
			} else {
				parts = append(parts, e.DestinoIATA)
			}
		}
		parts = append(parts, r.Destino.Ciudad+" ("+r.Destino.IATA+")")
		return strings.Join(parts, " - ")
	}
	return r.Tramo
}

func (r Ruta) GetTramos() []string {
	if r.OrigenIATA == "" || r.DestinoIATA == "" {
		return []string{r.Tramo}
	}

	var points []string
	points = append(points, r.Origen.Ciudad+" ("+r.OrigenIATA+")")
	for _, e := range r.Escalas {
		if e.Destino.Ciudad != "" {
			points = append(points, e.Destino.Ciudad+" ("+e.DestinoIATA+")")
		} else {
			points = append(points, e.DestinoIATA)
		}
	}
	points = append(points, r.Destino.Ciudad+" ("+r.DestinoIATA+")")

	var tramos []string
	for i := 0; i < len(points)-1; i++ {
		tramos = append(tramos, points[i]+" - "+points[i+1])
	}
	return tramos
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
