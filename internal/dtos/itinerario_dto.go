package dtos

type RutaView struct {
	Display string
	Origen  string
	Destino string
}

type ConnectionView struct {
	ID              string
	Tipo            string
	Ruta            RutaView
	RutaID          string
	Fecha           string
	Boleto          string
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

type TicketView struct {
	Boleto          string
	Scales          []ConnectionView
	EsDevolucion    bool
	EsModificacion  bool
	MontoDevolucion float64
	Moneda          string
}
