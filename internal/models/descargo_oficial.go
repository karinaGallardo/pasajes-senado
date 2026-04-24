package models

import (
	"strings"
	"time"
)

type DescargoOficial struct {
	BaseModel
	DescargoID string `gorm:"size:36;not null;uniqueIndex"`

	NroMemorandum     string `gorm:"size:100"`
	ObjetivoViaje     string `gorm:"type:text"`
	TipoTransporte    string `gorm:"size:100"` // AEREO, TERRESTRE, VEHICULO_OFICIAL
	PlacaVehiculo     string `gorm:"size:50"`
	ArchivoMemorandum string `gorm:"size:255"`

	InformeActividades          string `gorm:"type:text"`
	ResultadosViaje             string `gorm:"column:resultados_viaje;type:text"`
	ConclusionesRecomendaciones string `gorm:"column:conclusiones_recomendaciones;type:text"`

	NroBoletaDeposito string `gorm:"size:100"`
	DirigidoA         string `gorm:"size:255"`
	LugarViaje        string `gorm:"size:100;default:''"`

	FechaSalida  time.Time `gorm:"type:timestamp"`
	FechaRetorno time.Time `gorm:"type:timestamp"`

	Anexos                []AnexoDescargo               `gorm:"foreignKey:DescargoOficialID"`
	TransportesTerrestres []TransporteTerrestreDescargo `gorm:"foreignKey:DescargoOficialID"`
}

func (d DescargoOficial) HasChanges(other DescargoOficial) bool {
	return d.DescargoID != other.DescargoID ||
		d.NroMemorandum != other.NroMemorandum ||
		d.ObjetivoViaje != other.ObjetivoViaje ||
		d.LugarViaje != other.LugarViaje ||
		d.TipoTransporte != other.TipoTransporte ||
		d.PlacaVehiculo != other.PlacaVehiculo ||
		d.ArchivoMemorandum != other.ArchivoMemorandum ||
		d.InformeActividades != other.InformeActividades ||
		d.ResultadosViaje != other.ResultadosViaje ||
		d.ConclusionesRecomendaciones != other.ConclusionesRecomendaciones ||
		d.NroBoletaDeposito != other.NroBoletaDeposito ||
		d.DirigidoA != other.DirigidoA ||
		!d.FechaSalida.Equal(other.FechaSalida) ||
		!d.FechaRetorno.Equal(other.FechaRetorno)
}

func (DescargoOficial) TableName() string {
	return "descargos_oficiales"
}

func (d DescargoOficial) GetTipoTransporteDisplay() string {
	if d.TipoTransporte == "" {
		return "No especificado"
	}

	parts := strings.Split(d.TipoTransporte, ",")
	var displays []string

	for _, p := range parts {
		p = strings.TrimSpace(p)
		switch p {
		case "AEREO":
			displays = append(displays, "Aéreo")
		case "TERRESTRE_PUBLICO":
			displays = append(displays, "Público Terrestre")
		case "VEHICULO_OFICIAL":
			res := "Vehículo Oficial"
			if d.PlacaVehiculo != "" {
				res += " (Placa: " + d.PlacaVehiculo + ")"
			}
			displays = append(displays, res)
		default:
			displays = append(displays, p)
		}
	}

	return strings.Join(displays, ", ")
}

func (d DescargoOficial) HasTransportType(t string) bool {
	parts := strings.Split(d.TipoTransporte, ",")
	for _, p := range parts {
		if strings.TrimSpace(p) == t {
			return true
		}
	}
	return false
}

func (d DescargoOficial) GetTransporteList() []string {
	if d.TipoTransporte == "" {
		return []string{}
	}
	parts := strings.Split(d.TipoTransporte, ",")
	seen := make(map[string]bool)
	var res []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t == "TERRESTRE" {
			t = "TERRESTRE_PUBLICO"
		}
		if t != "" && !seen[t] {
			seen[t] = true
			res = append(res, t)
		}
	}
	return res
}
