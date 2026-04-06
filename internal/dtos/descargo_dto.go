package dtos

import (
	"mime/multipart"
	"sistema-pasajes/internal/models"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

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

	// Transporte Terrestre Público
	TransporteTerrestreFecha   []string `form:"terrestre_fecha[]"`
	TransporteTerrestreFactura []string `form:"terrestre_factura[]"`
	TransporteTerrestreImporte []string `form:"terrestre_importe[]"`
	TransporteTerrestreTipo    []string `form:"terrestre_tipo[]"`

	// Detalles del Tramo (FV-05) - Arreglos paralelos para conexiones
	TramoTipo            []string                `form:"tramo_tipo[]"` // IDA_ORIGINAL, IDA_REPRO, VUELTA_ORIGINAL, VUELTA_REPRO
	TramoIndex           []string                `form:"tramo_index[]"`
	TramoID              []string                `form:"tramo_id[]"`
	TramoRutaID          []string                `form:"tramo_ruta_id[]"`
	TramoRutaNombre      []string                `form:"tramo_ruta_nombre[]"`
	TramoOrigenIATA      []string                `form:"tramo_origen_iata[]"`
	TramoDestinoIATA     []string                `form:"tramo_destino_iata[]"`
	TramoFecha           []string                `form:"tramo_fecha[]"`
	TramoBillete         []string                `form:"tramo_billete[]"`
	TramoVuelo           []string                `form:"tramo_vuelo[]"`
	TramoPaseNumero      []string                `form:"tramo_pase_numero[]"`
	TramoDevolucion      []string                `form:"tramo_devolucion[]"`
	TramoModificacion    []string                `form:"tramo_modificacion[]"`
	TramoMontoDevolucion []string                `form:"tramo_monto_devolucion[]"`
	TramoMoneda          []string                `form:"tramo_moneda[]"`
	TramoPasajeID        []string                `form:"tramo_pasaje_id[]"`
	TramoSolicitudItemID []string                `form:"tramo_solicitud_item_id[]"`
	TramoPaseArchivo     []*multipart.FileHeader `form:"tramo_archivo[]"`
}

// TramoRowDTO representa una fila de itinerario ya procesada y tipada.
type TramoRowDTO struct {
	ID              string
	Tipo            string
	RutaID          string
	RutaNombre      string
	OrigenIATA      string
	DestinoIATA     string
	Fecha           string
	Billete         string
	Vuelo           string
	PaseNumero      string
	MontoDevolucion float64
	Moneda          string
	PasajeID        string
	SolicitudItemID string
	EsDevolucion    bool
	EsModificacion  bool
	ArchivoPath     string
	Seq             int
}

// ToTramoRows transforma los arreglos paralelos en una lista de objetos estructurados.
func (r *CreateDescargoRequest) ToTramoRows(archivoPaths []string) []TramoRowDTO {
	count := len(r.TramoTipo)
	rows := make([]TramoRowDTO, 0, count)

	// Mapas para checkboxes
	devoMap := make(map[string]bool)
	for _, id := range r.TramoDevolucion {
		devoMap[id] = true
	}
	modMap := make(map[string]bool)
	for _, id := range r.TramoModificacion {
		modMap[id] = true
	}

	for i := range count {
		rawID := ""
		if i < len(r.TramoID) {
			rawID = strings.TrimSpace(r.TramoID[i])
		}
		// Si es una fila nueva guiada por el index del frontend
		if (rawID == "" || strings.HasPrefix(rawID, "new_")) && i < len(r.TramoIndex) {
			rawID = r.TramoIndex[i]
		}

		get := func(arr []string, idx int) string {
			if idx < len(arr) {
				return arr[idx]
			}
			return ""
		}

		monto, _ := strconv.ParseFloat(get(r.TramoMontoDevolucion, i), 64)

		rows = append(rows, TramoRowDTO{
			ID:              rawID,
			Tipo:            get(r.TramoTipo, i),
			RutaID:          get(r.TramoRutaID, i),
			RutaNombre:      get(r.TramoRutaNombre, i),
			OrigenIATA:      get(r.TramoOrigenIATA, i),
			DestinoIATA:     get(r.TramoDestinoIATA, i),
			Fecha:           get(r.TramoFecha, i),
			Billete:         strings.ToUpper(strings.TrimSpace(get(r.TramoBillete, i))),
			Vuelo:           get(r.TramoVuelo, i),
			PaseNumero:      get(r.TramoPaseNumero, i),
			MontoDevolucion: monto,
			Moneda:          get(r.TramoMoneda, i),
			PasajeID:        get(r.TramoPasajeID, i),
			SolicitudItemID: get(r.TramoSolicitudItemID, i),
			EsDevolucion:    devoMap[rawID],
			EsModificacion:  modMap[rawID],
			ArchivoPath:     get(archivoPaths, i),
			Seq:             i + 1,
		})
	}
	return rows
}

// Bind realiza tanto el binding estándar de Gin como la corrección manual de arreglos paralelos.
func (r *CreateDescargoRequest) Bind(c *gin.Context) error {
	// 1. Binding estándar de Gin
	if err := c.ShouldBind(r); err != nil {
		return err
	}

	// 2. Corrección manual de arreglos (ManualBind)
	r.TramoTipo = c.PostFormArray("tramo_tipo[]")
	r.TramoID = c.PostFormArray("tramo_id[]")
	r.TramoRutaID = c.PostFormArray("tramo_ruta_id[]")
	r.TramoFecha = c.PostFormArray("tramo_fecha[]")
	r.TramoBillete = c.PostFormArray("tramo_billete[]")
	r.TramoVuelo = c.PostFormArray("tramo_vuelo[]")
	r.TramoPaseNumero = c.PostFormArray("tramo_pase_numero[]")
	r.TramoDevolucion = c.PostFormArray("tramo_devolucion[]")
	r.TramoModificacion = c.PostFormArray("tramo_modificacion[]")
	r.TramoMontoDevolucion = c.PostFormArray("tramo_monto_devolucion[]")
	r.TramoMoneda = c.PostFormArray("tramo_moneda[]")
	r.TramoPasajeID = c.PostFormArray("tramo_pasaje_id[]")
	r.TramoSolicitudItemID = c.PostFormArray("tramo_solicitud_item_id[]")
	r.TramoOrigenIATA = c.PostFormArray("tramo_origen_iata[]")
	r.TramoDestinoIATA = c.PostFormArray("tramo_destino_iata[]")

	return nil
}

type ItinerarioTramos struct {
	Billete string                 // Nro de Billete / E-Ticket
	Tramos  []models.DescargoTramo // Tramos agrupados por este billete
}

// DescargoShowData contiene toda la información necesaria para renderizar el detalle del descargo
type DescargoShowData struct {
	Descargo *models.Descargo
	Ida      []models.DescargoTramo
	Vuelta   []models.DescargoTramo
}

// DescargoEditData contiene la información estructurada para el formulario de edición
type DescargoEditData struct {
	Descargo  *models.Descargo
	Solicitud *models.Solicitud
	Ida       []models.DescargoTramo
	Vuelta    []models.DescargoTramo
}
