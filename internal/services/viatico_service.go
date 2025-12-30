package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strconv"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"gorm.io/gorm"
)

type ViaticoService struct {
	db            *gorm.DB
	repo          repositories.ViaticoRepository
	solicitudRepo *repositories.SolicitudRepository
	configService *ConfiguracionService
}

func NewViaticoService(db *gorm.DB) *ViaticoService {
	return &ViaticoService{
		db:            db,
		repo:          repositories.NewViaticoRepository(db),
		solicitudRepo: repositories.NewSolicitudRepository(db),
		configService: NewConfiguracionService(db),
	}
}

type DetalleViaticoInput struct {
	FechaDesde time.Time
	FechaHasta time.Time
	Dias       float64
	Lugar      string
	MontoDia   float64
	Porcentaje int
}

func (s *ViaticoService) RegistrarViatico(solicitudID string, detalles []DetalleViaticoInput, tieneGastosRep bool, usuarioID string) (*models.Viatico, error) {
	var total float64
	var viaticoDetalles []models.DetalleViatico

	for _, d := range detalles {
		dailyRate := d.MontoDia * (float64(d.Porcentaje) / 100.0)
		subTotal := dailyRate * d.Dias

		total += subTotal

		viaticoDetalles = append(viaticoDetalles, models.DetalleViatico{
			FechaDesde: d.FechaDesde,
			FechaHasta: d.FechaHasta,
			Dias:       d.Dias,
			Lugar:      d.Lugar,
			MontoDia:   d.MontoDia,
			Porcentaje: d.Porcentaje,
			SubTotal:   subTotal,
		})
	}

	rcIvaStr := s.configService.GetValue("IMPUESTO_RC_IVA")
	rcIvaRate := 0.13
	if rcIvaStr != "" {
		if val, err := strconv.ParseFloat(rcIvaStr, 64); err == nil {
			rcIvaRate = val
			if rcIvaRate > 1 {
				rcIvaRate = rcIvaRate / 100.0
			}
		}
	}

	rcIva := total * rcIvaRate
	liquido := total - rcIva

	var gastos, retGastos, liqGastos float64
	if tieneGastosRep {
		gastos = total * 0.25
		retGastos = gastos * rcIvaRate
		liqGastos = gastos - retGastos
	}

	viatico := &models.Viatico{
		SolicitudID:          solicitudID,
		UsuarioID:            usuarioID,
		FechaAsignacion:      time.Now(),
		MontoTotal:           total,
		MontoRC_IVA:          rcIva,
		MontoLiquido:         liquido,
		TieneGastosRep:       tieneGastosRep,
		MontoGastosRep:       gastos,
		MontoRetencionGastos: retGastos,
		MontoLiquidoGastos:   liqGastos,
		Estado:               "ASIGNADO",
		Detalles:             viaticoDetalles,
		Codigo:               generateViaticoCode(),
		Glosa:                "Asignación automática de viáticos",
	}

	if err := s.repo.Create(viatico); err != nil {
		return nil, err
	}

	return viatico, nil
}

func (s *ViaticoService) FindBySolicitud(solicitudID string) ([]models.Viatico, error) {
	return s.repo.FindBySolicitudID(solicitudID)
}

func (s *ViaticoService) FindByID(id string) (*models.Viatico, error) {
	return s.repo.FindByID(id)
}

func generateViaticoCode() string {
	id, _ := gonanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ", 8)
	return "V-" + id
}
