package models

type TipoCargoPasaje string

const (
	CargoEmision     TipoCargoPasaje = "EMISION"
	CargoCambioFecha TipoCargoPasaje = "CAMBIO_FECHA"
	CargoCambioRuta  TipoCargoPasaje = "CAMBIO_RUTA"
	CargoCambioVuelo TipoCargoPasaje = "CAMBIO_VUELO"
	CargoOtros       TipoCargoPasaje = "OTROS"
)

// PasajeCargo representa cargos asociados al pasaje facturados por la agencia o aerolínea
// vinculados a un pasaje específico (Ej: Emisión, Cambios, etc.)
type PasajeCargo struct {
	BaseModel
	PasajeID string  `gorm:"size:36;not null;index"`
	Tipo     string  `gorm:"size:50;not null;index"` // Se guarda el código de TipoCargoPasaje
	Factura  string  `gorm:"size:50;index"`
	Monto    float64 `gorm:"type:decimal(10,2);default:0"`
	Archivo  string  `gorm:"size:255;default:''"`
	Glosa    string  `gorm:"type:text"`
}

func (PasajeCargo) TableName() string {
	return "pasaje_cargos"
}

func (c PasajeCargo) GetTipoDisplay() string {
	switch TipoCargoPasaje(c.Tipo) {
	case CargoEmision:
		return "Servicio de Emisión"
	case CargoCambioFecha:
		return "Cambio de Fecha"
	case CargoCambioRuta:
		return "Cambio de Ruta"
	case CargoCambioVuelo:
		return "Cambio de Vuelo"
	case CargoOtros:
		return "Otros Cargos"
	default:
		return c.Tipo
	}
}
