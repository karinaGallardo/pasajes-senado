package dtos

type ReprogramarPasajeRequest struct {
	PasajeAnteriorID string `form:"pasaje_anterior_id" binding:"required"`
	Costo            string `form:"costo" binding:"required"`
	CostoPenalidad   string `form:"costo_penalidad"`
	FechaVuelo       string `form:"fecha_vuelo" binding:"required"`
	Ruta             string `form:"ruta" binding:"required"`
	NumeroVuelo      string `form:"numero_vuelo" binding:"required"`
	NumeroBoleto     string `form:"numero_boleto" binding:"required"`
	Glosa            string `form:"glosa"`
	AerolineaID      string `form:"aerolinea_id"`
	AgenciaID        string `form:"agencia_id" binding:"required"`
	NumeroFactura    string `form:"numero_factura"`
	CodigoReserva    string `form:"codigo_reserva"`
}
