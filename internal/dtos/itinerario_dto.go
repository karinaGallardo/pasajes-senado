package dtos

import "strings"

type RutaView struct {
	Display string
	Origen  string
	Destino string
}

type TramoView struct {
	ID              string
	Tipo            string
	Ruta            RutaView
	RutaID          string
	Fecha           string
	Billete         string
	Pase            string
	Archivo         string
	EsDevolucion    bool
	EsModificacion  bool
	MontoDevolucion float64
	Moneda          string
	PasajeID        string
	SolicitudItemID string
}

func (t TramoView) IsOriginal() bool {
	upper := strings.ToUpper(t.Tipo)
	return strings.HasSuffix(upper, "_ORIGINAL")
}

func (t TramoView) IsReprogramacion() bool {
	upper := strings.ToUpper(t.Tipo)
	return strings.HasSuffix(upper, "_REPRO") || strings.HasSuffix(upper, "_REPROG")
}

type ItinerarioView struct {
	Billete string
	Tramos  []TramoView
}
