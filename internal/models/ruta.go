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
	if r.Origen.IATA != "" && r.Destino.IATA != "" {
		parts := []string{r.Origen.GetNombreCorto()}
		for _, e := range r.Escalas {
			if e.Destino.IATA != "" {
				parts = append(parts, e.Destino.GetNombreCorto())
			} else {
				parts = append(parts, e.DestinoIATA)
			}
		}
		parts = append(parts, r.Destino.GetNombreCorto())
		return strings.Join(parts, " - ")
	}
	return r.Tramo
}

type TramoLeg struct {
	OrigenIATA    string
	DestinoIATA   string
	OrigenCiudad  string
	DestinoCiudad string
}

func (r Ruta) GetTramosItems() []TramoLeg {
	// 1. Recolectar todos los puntos de la ruta con sus ciudades
	type point struct {
		iata   string
		ciudad string
	}

	var points []point
	points = append(points, point{iata: r.OrigenIATA, ciudad: r.Origen.Ciudad})

	for _, e := range r.Escalas {
		points = append(points, point{iata: e.DestinoIATA, ciudad: e.Destino.Ciudad})
	}
	points = append(points, point{iata: r.DestinoIATA, ciudad: r.Destino.Ciudad})

	// 2. Generar los tramos pareados
	var tramos []TramoLeg
	for i := 0; i < len(points)-1; i++ {
		p1, p2 := points[i], points[i+1]
		tramos = append(tramos, TramoLeg{
			OrigenIATA:    p1.iata,
			DestinoIATA:   p2.iata,
			OrigenCiudad:  p1.ciudad,
			DestinoCiudad: p2.ciudad,
		})
	}
	return tramos
}

func (r TramoLeg) GetLabel() string {
	orig := r.OrigenIATA
	if r.OrigenCiudad != "" {
		orig = r.OrigenCiudad + " (" + r.OrigenIATA + ")"
	}
	dest := r.DestinoIATA
	if r.DestinoCiudad != "" {
		dest = r.DestinoCiudad + " (" + r.DestinoIATA + ")"
	}
	return orig + " - " + dest
}

func (Ruta) TableName() string { return "rutas" }
