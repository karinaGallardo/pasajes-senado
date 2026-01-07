package dtos

type CreatePasajeRequest struct {
	Costo         string `form:"costo" binding:"required"`
	FechaVuelo    string `form:"fecha_vuelo" binding:"required"`
	Aerolinea     string `form:"aerolinea"`
	NumeroVuelo   string `form:"numero_vuelo" binding:"required"`
	Ruta          string `form:"ruta" binding:"required"`
	CodigoReserva string `form:"codigo_reserva" binding:"required"`
	NumeroBoleto  string `form:"numero_boleto" binding:"required"`
	AgenciaID     string `form:"agencia_id" binding:"required"`
	NumeroFactura string `form:"numero_factura"`
	Glosa         string `form:"glosa"`
}
