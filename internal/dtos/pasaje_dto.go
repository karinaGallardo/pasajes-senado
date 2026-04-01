package dtos

type CreatePasajeRequest struct {
	SolicitudItemID string `form:"solicitud_item_id" binding:"required"`
	Costo           string `form:"costo" binding:"required"`
	FechaVuelo      string `form:"fecha_vuelo" binding:"required"`
	FechaEmision    string `form:"fecha_emision"`
	AerolineaID     string `form:"aerolinea_id"`
	NumeroVuelo     string `form:"numero_vuelo" binding:"required"`
	RutaID          string `form:"ruta_id"`
	CodigoReserva   string `form:"codigo_reserva"`
	NumeroBillete   string `form:"numero_billete" binding:"required"`
	AgenciaID       string `form:"agencia_id" binding:"required"`
	NumeroFactura   string `form:"numero_factura"`
	Glosa           string `form:"glosa"`
}

type UpdatePasajeRequest struct {
	ID            string `form:"id" binding:"required"`
	Costo         string `form:"costo" binding:"required"`
	FechaVuelo    string `form:"fecha_vuelo" binding:"required"`
	FechaEmision  string `form:"fecha_emision"`
	AerolineaID   string `form:"aerolinea_id"`
	NumeroVuelo   string `form:"numero_vuelo" binding:"required"`
	RutaID        string `form:"ruta_id"`
	CodigoReserva string `form:"codigo_reserva"`
	NumeroBillete string `form:"numero_billete" binding:"required"`
	AgenciaID     string `form:"agencia_id" binding:"required"`
	NumeroFactura string `form:"numero_factura"`
	Glosa         string `form:"glosa"`
}

type UpdatePasajeStatusRequest struct {
	ID     string `form:"id" binding:"required"`
	Status string `form:"status" binding:"required"`
}

type DevolverPasajeRequest struct {
	PasajeID       string `form:"pasaje_id" binding:"required"`
	Glosa          string `form:"glosa"`
	CostoPenalidad string `form:"costo_penalidad"`
}
