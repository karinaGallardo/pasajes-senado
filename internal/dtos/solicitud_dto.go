package dtos

type CreateSolicitudRequest struct {
	TipoSolicitudID string `form:"tipo_solicitud_id" binding:"required"`
	AmbitoViajeID   string `form:"ambito_viaje_id" binding:"required"`
	TargetUserID    string `form:"target_user_id"`

	// Deprecated: use TipoItinerarioID
	TipoItinerarioCode string `form:"tipo_itinerario"`

	OrigenCode        string `form:"origen" binding:"required"`
	DestinoCode       string `form:"destino" binding:"required"`
	FechaSalida       string `form:"fecha_salida" binding:"required"`
	FechaRetorno      string `form:"fecha_retorno"`
	Motivo            string `form:"motivo" binding:"required"`
	AerolineaSugerida string `form:"aerolinea"`
}

type UpdateSolicitudRequest struct {
	TipoSolicitudID   string `form:"tipo_solicitud_id" binding:"required"`
	AmbitoViajeID     string `form:"ambito_viaje_id" binding:"required"`
	TipoItinerarioID  string `form:"tipo_itinerario_id" binding:"required"`
	OrigenCod         string `form:"origen_cod" binding:"required"`
	DestinoCod        string `form:"destino_cod" binding:"required"`
	FechaSalida       string `form:"fecha_salida" binding:"required"`
	FechaRetorno      string `form:"fecha_retorno"`
	Motivo            string `form:"motivo" binding:"required"`
	AerolineaSugerida string `form:"aerolinea_sugerida"`
}
