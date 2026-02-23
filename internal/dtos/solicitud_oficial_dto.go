package dtos

type TramoOficialRequest struct {
	ID          string `json:"id"`
	OrigenIATA  string `json:"origen"`
	DestinoIATA string `json:"destino"`
	FechaSalida string `json:"fecha_salida"`
	Tipo        string `json:"tipo"` // IDA o VUELTA
	Estado      string `json:"estado"`
}

type CreateSolicitudOficialRequest struct {
	AmbitoViajeCodigo string                `form:"ambito_viaje_codigo" binding:"required"`
	TargetUserID      string                `form:"target_user_id"`
	Motivo            string                `form:"motivo"`
	Autorizacion      string                `form:"autorizacion"`
	AerolineaSugerida string                `form:"aerolinea"`
	Tramos            []TramoOficialRequest `form:"-"`
}
