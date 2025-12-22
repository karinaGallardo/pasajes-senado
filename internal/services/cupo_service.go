package services

import (
	"fmt"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"strings"
	"time"
)

type CupoService struct{}

func NewCupoService() *CupoService {
	return &CupoService{}
}

type InfoCupo struct {
	Mensaje      string
	EsDisponible bool
	Usados       int
	Limite       int
}

func (s *CupoService) CalcularCupo(usuarioID string, fecha time.Time) (InfoCupo, error) {
	mes := int(fecha.Month())
	anio := fecha.Year()

	var count int64

	err := configs.DB.Model(&models.Solicitud{}).
		Joins("JOIN tipo_solicitudes ON tipo_solicitudes.id = solicitudes.tipo_solicitud_id").
		Joins("JOIN concepto_viajes ON concepto_viajes.id = tipo_solicitudes.concepto_viaje_id").
		Where("solicitudes.usuario_id = ?", usuarioID).
		Where("concepto_viajes.codigo = ?", "DERECHO").
		Where("EXTRACT(MONTH FROM solicitudes.fecha_salida) = ?", mes).
		Where("EXTRACT(YEAR FROM solicitudes.fecha_salida) = ?", anio).
		Where("solicitudes.estado != ?", "RECHAZADO").
		Count(&count).Error

	if err != nil {
		return InfoCupo{}, err
	}

	usados := int(count)
	limite := 4

	disponible := usados < limite

	nombreMes := strings.ToUpper(MESES_ES[fecha.Month()-1])
	mensaje := fmt.Sprintf("%s - PASAJE %d", nombreMes, usados+1)

	if !disponible {
		mensaje = fmt.Sprintf("%s - CUPO AGOTADO (%d/%d)", nombreMes, usados, limite)
	}

	return InfoCupo{
		Mensaje:      mensaje,
		EsDisponible: disponible,
		Usados:       usados,
		Limite:       limite,
	}, nil
}

var MESES_ES = []string{
	"Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio",
	"Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre",
}
