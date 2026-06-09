package dtos

type TramoExtraRequest struct {
	OrigenIATA   string `json:"origen"`
	DestinoIATA  string `json:"destino"`
	FechaSalida  string `json:"fecha"`
	AerolineaID  string `json:"aerolinea_id"`
	OpenTicketID string `json:"open_ticket_id"`
	PorConfirmar bool   `json:"por_confirmar"`
}

type CreateSolicitudRequest struct {
	ConceptoCodigo      string `form:"concepto_codigo"`
	TipoSolicitudCodigo string `form:"tipo_solicitud_codigo" binding:"required"`
	AmbitoViajeCodigo   string `form:"ambito_viaje_codigo" binding:"required"`
	TargetUserID        string `form:"target_user_id"`

	TipoItinerarioCodigo string `form:"tipo_itinerario_codigo"`
	TipoItinerario       string `form:"tipo_itinerario"`

	OrigenIdaIATA      string `form:"origen_ida"`
	DestinoVueltaIATA  string `form:"destino_vuelta"`
	FechaIda           string `form:"fecha_salida"`
	FechaVuelta        string `form:"fecha_retorno"`
	Motivo             string `form:"motivo"`
	AerolineaID        string `form:"aerolinea_id"`
	CupoDerechoItemID  string `form:"cupo_derecho_item_id"`
	ActiveTab          string `form:"active_tab"`
	Autorizacion       string `form:"autorizacion"`
	ReturnURL          string `form:"return_url"`
	SedeIATA           string `form:"sede_iata"`
	IdaAerolineaID     string `form:"ida_aerolinea_id"`
	VueltaAerolineaID  string `form:"vuelta_aerolinea_id"`
	IdaOpenTicketID    string `form:"ida_open_ticket_id"`
	VueltaOpenTicketID string `form:"vuelta_open_ticket_id"`
	IdaPorConfirmar    bool   `form:"ida_por_confirmar"`
	VueltaPorConfirmar bool   `form:"vuelta_por_confirmar"`
	SoloIda            bool   `form:"solo_ida"`
	SoloVuelta         bool   `form:"solo_vuelta"`
	TramosExtraJSON    string `form:"tramos_extra_json"`
}

type UpdateSolicitudRequest struct {
	TipoSolicitudCodigo  string `form:"tipo_solicitud_codigo" binding:"required"`
	AmbitoViajeCodigo    string `form:"ambito_viaje_codigo" binding:"required"`
	TipoItinerarioCodigo string `form:"tipo_itinerario_codigo"`
	OrigenIdaIATA        string `form:"origen_ida_cod"`
	DestinoVueltaIATA    string `form:"destino_vuelta_cod"`
	FechaIda             string `form:"fecha_salida"`
	FechaVuelta          string `form:"fecha_retorno"`
	Motivo               string `form:"motivo"`
	AerolineaID          string `form:"aerolinea_id"`
	ActiveTab            string `form:"active_tab"`
	ReturnURL            string `form:"return_url"`
	SedeIATA             string `form:"sede_iata"`
	IdaAerolineaID       string `form:"ida_aerolinea_id"`
	VueltaAerolineaID    string `form:"vuelta_aerolinea_id"`
	IdaOpenTicketID      string `form:"ida_open_ticket_id"`
	VueltaOpenTicketID   string `form:"vuelta_open_ticket_id"`
	IdaPorConfirmar      bool   `form:"ida_por_confirmar"`
	VueltaPorConfirmar   bool   `form:"vuelta_por_confirmar"`
	SoloIda              bool   `form:"solo_ida"`
	SoloVuelta           bool   `form:"solo_vuelta"`
}
