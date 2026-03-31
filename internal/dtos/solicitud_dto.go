package dtos

type CreateSolicitudRequest struct {
	ConceptoCodigo      string `form:"concepto_codigo"`
	TipoSolicitudCodigo string `form:"tipo_solicitud_codigo" binding:"required"`
	AmbitoViajeCodigo   string `form:"ambito_viaje_codigo" binding:"required"`
	TargetUserID        string `form:"target_user_id"`

	TipoItinerarioCodigo string `form:"tipo_itinerario_codigo"`
	TipoItinerario       string `form:"tipo_itinerario"`

	OrigenIdaIATA      string `form:"origen_ida" binding:"required"`
	DestinoVueltaIATA  string `form:"destino_vuelta" binding:"required"`
	FechaIda           string `form:"fecha_salida"`
	FechaVuelta        string `form:"fecha_retorno"`
	Motivo             string `form:"motivo"`
	AerolineaSugerida  string `form:"aerolinea"`
	CupoDerechoItemID  string `form:"cupo_derecho_item_id"`
	ActiveTab          string `form:"active_tab"`
	Autorizacion       string `form:"autorizacion"`
	ReturnURL          string `form:"return_url"`
	IdaPorConfirmar    bool   `form:"ida_por_confirmar"`
	VueltaPorConfirmar bool   `form:"vuelta_por_confirmar"`
}

type UpdateSolicitudRequest struct {
	TipoSolicitudCodigo  string `form:"tipo_solicitud_codigo" binding:"required"`
	AmbitoViajeCodigo    string `form:"ambito_viaje_codigo" binding:"required"`
	TipoItinerarioCodigo string `form:"tipo_itinerario_codigo"`
	OrigenIdaIATA        string `form:"origen_ida_cod" binding:"required"`
	DestinoVueltaIATA    string `form:"destino_vuelta_cod" binding:"required"`
	FechaIda             string `form:"fecha_salida"`
	FechaVuelta          string `form:"fecha_retorno"`
	Motivo               string `form:"motivo"`
	AerolineaSugerida    string `form:"aerolinea_sugerida"`
	ActiveTab            string `form:"active_tab"`
	ReturnURL            string `form:"return_url"`
	IdaPorConfirmar      bool   `form:"ida_por_confirmar"`
	VueltaPorConfirmar   bool   `form:"vuelta_por_confirmar"`
}
