package dtos

import "mime/multipart"

type CreateDescargoRequest struct {
	SolicitudID        string `form:"solicitud_id" binding:"required"`
	InformeActividades string `form:"informe_actividades"`
	Observaciones      string `form:"observaciones"`

	// Detalles Itinerario (FV-05) - Arreglos paralelos para conexiones
	ItinTipo        []string                `form:"itin_tipo[]"` // IDA_ORIGINAL, IDA_REPRO, VUELTA_ORIGINAL, VUELTA_REPRO
	ItinRuta        []string                `form:"itin_ruta[]"`
	ItinFecha       []string                `form:"itin_fecha[]"`
	ItinBoleto      []string                `form:"itin_boleto[]"`
	ItinPaseNumero  []string                `form:"itin_pase_numero[]"`
	ItinPaseArchivo []*multipart.FileHeader `form:"itin_archivo[]"`
}
