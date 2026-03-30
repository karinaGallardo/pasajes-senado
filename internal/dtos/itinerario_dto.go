package dtos

type ConnectionView struct {
	ID              string
	Ruta            string
	RutaID          string
	Fecha           string
	Boleto          string
	Index           string
	Pase            string
	Archivo         string
	EsDevolucion    bool
	EsModificacion  bool
	MontoDevolucion float64
	CostoPasaje     float64
	TotalScales     int
	IsFirstScale    bool
	Orden           int
}

type TicketView struct {
	Boleto          string
	Scales          []ConnectionView
	EsDevolucion    bool
	EsModificacion  bool
	MontoDevolucion float64
	CostoPasaje     float64
}
