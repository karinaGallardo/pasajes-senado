package models

type DescargoOficial struct {
	BaseModel
	DescargoID string `gorm:"size:36;not null;uniqueIndex"`

	NroMemorandum  string `gorm:"size:100"`
	ObjetivoViaje  string `gorm:"type:text"`
	TipoTransporte string `gorm:"size:50"` // AEREO, TERRESTRE, VEHICULO_OFICIAL
	PlacaVehiculo  string `gorm:"size:50"`

	InformeActividades          string `gorm:"type:text"`
	ResultadosViaje             string `gorm:"column:resultados_viaje;type:text"`
	ConclusionesRecomendaciones string `gorm:"column:conclusiones_recomendaciones;type:text"`

	MontoDevolucion   float64 `gorm:"type:decimal(10,2);default:0"`
	NroBoletaDeposito string  `gorm:"size:100"`
	DirigidoA         string  `gorm:"size:255"`

	Anexos                []AnexoDescargo              `gorm:"foreignKey:DescargoOficialID"`
	TransportesTerrestres []TransporteTerrestreDescargo `gorm:"foreignKey:DescargoOficialID"`
}

func (DescargoOficial) TableName() string {
	return "descargos_oficiales"
}

func (d DescargoOficial) GetTipoTransporteDisplay() string {
	switch d.TipoTransporte {
	case "AEREO":
		return "Aéreo"
	case "TERRESTRE_PUBLICO":
		return "Público Terrestre"
	case "VEHICULO_OFICIAL":
		res := "Vehículo Oficial"
		if d.PlacaVehiculo != "" {
			res += " (Placa: " + d.PlacaVehiculo + ")"
		}
		return res
	default:
		return d.TipoTransporte
	}
}
