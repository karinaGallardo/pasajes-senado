package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strconv"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

type ViaticoService struct {
	repo          *repositories.ViaticoRepository
	solicitudRepo *repositories.SolicitudRepository
	catRepo       *repositories.CategoriaViaticoRepository
	configService *ConfiguracionService
}

func NewViaticoService() *ViaticoService {
	return &ViaticoService{
		repo:          repositories.NewViaticoRepository(),
		solicitudRepo: repositories.NewSolicitudRepository(),
		catRepo:       repositories.NewCategoriaViaticoRepository(),
		configService: NewConfiguracionService(),
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

func (s *ViaticoService) RegistrarViatico(ctx context.Context, solicitudID string, detalles []DetalleViaticoInput, tieneGastosRep bool, usuarioID string) (*models.Viatico, error) {
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

	rcivaPorcentajeStr := s.configService.GetValue(ctx, "IMPUESTO_RC_IVA")
	rcIvaRate := 0.13
	if rcivaPorcentajeStr != "" {
		if val, err := strconv.ParseFloat(rcivaPorcentajeStr, 64); err == nil {
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

	if err := s.repo.WithContext(ctx).Create(viatico); err != nil {
		return nil, err
	}

	return viatico, nil
}

func (s *ViaticoService) FindBySolicitud(ctx context.Context, solicitudID string) ([]models.Viatico, error) {
	return s.repo.WithContext(ctx).FindBySolicitudID(solicitudID)
}

func (s *ViaticoService) FindByID(ctx context.Context, id string) (*models.Viatico, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *ViaticoService) GetCategorias(ctx context.Context) ([]models.CategoriaViatico, error) {
	return s.catRepo.WithContext(ctx).FindAll()
}

func generateViaticoCode() string {
	id, _ := gonanoid.Generate("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ", 8)
	return "V-" + id
}
