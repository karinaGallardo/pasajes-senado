package dtos

type ReportFilterRequest struct {
	FechaDesde  string `form:"fecha_desde" json:"fecha_desde"`
	FechaHasta  string `form:"fecha_hasta" json:"fecha_hasta"`
	AerolineaID string `form:"aerolinea_id" json:"aerolinea_id"`
	AgenciaID   string `form:"agencia_id" json:"agencia_id"`
	Concepto    string `form:"concepto" json:"concepto"` // DERECHO, OFICIAL, ALL
	Estado      string `form:"estado" json:"estado"`     // EMITIDO, USADO, ANULADO, ALL
}
