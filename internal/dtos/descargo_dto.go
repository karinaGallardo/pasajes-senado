package dtos

type CreateDescargoRequest struct {
	SolicitudID        string `form:"solicitud_id" binding:"required"`
	FechaPresentacion  string `form:"fecha_presentacion" binding:"required"`
	MontoDevolucion    string `form:"monto_devolucion"`
	NumeroCite         string `form:"numero_cite"`
	InformeActividades string `form:"informe_actividades"`
	Observaciones      string `form:"observaciones"`

	DocTipo    []string `form:"doc_tipo[]"`
	DocNumero  []string `form:"doc_numero[]"`
	DocFecha   []string `form:"doc_fecha[]"`
	DocDetalle []string `form:"doc_detalle[]"`
}
