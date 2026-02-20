package dtos

type CreateSolicitudRequest struct {
	ConceptoCodigo      string `form:"concepto_codigo"`
	TipoSolicitudCodigo string `form:"tipo_solicitud_codigo" binding:"required"`
	AmbitoViajeCodigo   string `form:"ambito_viaje_codigo" binding:"required"`
	TargetUserID        string `form:"target_user_id"`

	TipoItinerarioCodigo string `form:"tipo_itinerario_codigo"`
	TipoItinerario       string `form:"tipo_itinerario"`

	OrigenIATA        string `form:"origen" binding:"required"`
	DestinoIATA       string `form:"destino" binding:"required"`
	FechaIda          string `form:"fecha_salida"`
	FechaVuelta       string `form:"fecha_retorno"`
	Motivo            string `form:"motivo"`
	AerolineaSugerida string `form:"aerolinea"`
	CupoDerechoItemID string `form:"cupo_derecho_item_id"`
	ActiveTab         string `form:"active_tab"`
	Autorizacion      string `form:"autorizacion"`
	ReturnURL         string `form:"return_url"`
}

type UpdateSolicitudRequest struct {
	TipoSolicitudCodigo  string `form:"tipo_solicitud_codigo" binding:"required"`
	AmbitoViajeCodigo    string `form:"ambito_viaje_codigo" binding:"required"`
	TipoItinerarioCodigo string `form:"tipo_itinerario_codigo" binding:"required"`
	OrigenIATA           string `form:"origen_cod" binding:"required"`
	DestinoIATA          string `form:"destino_cod" binding:"required"`
	FechaIda             string `form:"fecha_salida"`
	FechaVuelta          string `form:"fecha_retorno"`
	Motivo               string `form:"motivo"`
	AerolineaSugerida    string `form:"aerolinea_sugerida"`
	ActiveTab            string `form:"active_tab"`
	ReturnURL            string `form:"return_url"`
}
