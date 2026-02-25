package dtos

import "mime/multipart"

type CreateDescargoRequest struct {
	SolicitudID                 string `form:"solicitud_id" binding:"required"`
	InformeActividades          string `form:"informe_actividades"`
	ObjetivoViaje               string `form:"objetivo_viaje"`
	ResultadosViaje             string `form:"resultados_viaje"`
	ConclusionesRecomendaciones string `form:"conclusiones_recomendaciones"`
	Observaciones               string `form:"observaciones"`
	DirigidoA                   string `form:"dirigido_a"`

	// Informe PV-06
	NroMemorandum     string  `form:"nro_memorandum"`
	TipoTransporte    string  `form:"tipo_transporte"` // AEREO, TERRESTRE, VEHICULO_OFICIAL
	PlacaVehiculo     string  `form:"placa_vehiculo"`
	MontoDevolucion   float64 `form:"monto_devolucion"`
	NroBoletaDeposito string  `form:"nro_boleta_deposito"`

	// Detalles Itinerario (FV-05) - Arreglos paralelos para conexiones
	ItinTipo         []string                `form:"itin_tipo[]"` // IDA_ORIGINAL, IDA_REPRO, VUELTA_ORIGINAL, VUELTA_REPRO
	ItinIndex        []string                `form:"itin_index[]"`
	ItinRuta         []string                `form:"itin_ruta[]"`
	ItinFecha        []string                `form:"itin_fecha[]"`
	ItinBoleto       []string                `form:"itin_boleto[]"`
	ItinPaseNumero   []string                `form:"itin_pase_numero[]"`
	ItinDevolucion   []string                `form:"itin_devolucion[]"`
	ItinModificacion []string                `form:"itin_modificacion[]"`
	ItinPaseArchivo  []*multipart.FileHeader `form:"itin_archivo[]"`
}
