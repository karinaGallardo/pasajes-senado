package dtos

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
	Orden           int
	PasajeID        string
	SolicitudItemID string
}

type ItinerarioView struct {
	Billete string
	Tramos  []TramoView
}
