package dtos

type CreateSolicitudRequest struct {
	ConceptoID      string `form:"concepto_id"`
	TipoSolicitudID string `form:"tipo_solicitud_id" binding:"required"`
	AmbitoViajeID   string `form:"ambito_viaje_id" binding:"required"`
	TargetUserID    string `form:"target_user_id"`

	TipoItinerarioID string `form:"tipo_itinerario_id"`

	OrigenIATA        string `form:"origen" binding:"required"`
	DestinoIATA       string `form:"destino" binding:"required"`
	FechaIda          string `form:"fecha_salida"`
	FechaVuelta       string `form:"fecha_retorno"`
	Motivo            string `form:"motivo"`
	AerolineaSugerida string `form:"aerolinea"`
	VoucherID         string `form:"voucher_id"`
	ActiveTab         string `form:"active_tab"`
}

type UpdateSolicitudRequest struct {
	TipoSolicitudID   string `form:"tipo_solicitud_id" binding:"required"`
	AmbitoViajeID     string `form:"ambito_viaje_id" binding:"required"`
	TipoItinerarioID  string `form:"tipo_itinerario_id" binding:"required"`
	OrigenIATA        string `form:"origen_cod" binding:"required"`
	DestinoIATA       string `form:"destino_cod" binding:"required"`
	FechaIda          string `form:"fecha_salida"`
	FechaVuelta       string `form:"fecha_retorno"`
	Motivo            string `form:"motivo"`
	AerolineaSugerida string `form:"aerolinea_sugerida"`
	ActiveTab         string `form:"active_tab"`
}
