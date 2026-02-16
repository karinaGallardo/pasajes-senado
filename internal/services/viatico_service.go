package services

import (
	"context"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strconv"
	"time"
)

type ViaticoService struct {
	repo          *repositories.ViaticoRepository
	solicitudRepo *repositories.SolicitudRepository
	catRepo       *repositories.CategoriaViaticoRepository
	zonaRepo      *repositories.ZonaViaticoRepository
	configService *ConfiguracionService
}

func NewViaticoService() *ViaticoService {
	return &ViaticoService{
		repo:          repositories.NewViaticoRepository(),
		solicitudRepo: repositories.NewSolicitudRepository(),
		catRepo:       repositories.NewCategoriaViaticoRepository(),
		zonaRepo:      repositories.NewZonaViaticoRepository(),
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

func (s *ViaticoService) RegistrarViatico(ctx context.Context, solicitudID string, req dtos.CreateViaticoRequest, userID string) (*models.Viatico, error) {
	dias := utils.ParseFloat(req.Dias)
	montoDia := utils.ParseFloat(req.MontoDia)
	porcentaje, _ := strconv.Atoi(req.Porcentaje)
	tieneGastosRep := req.GastosRep == "on"

	layout := "2006-01-02"
	fechaDesde := utils.ParseDate(layout, req.FechaDesde)
	fechaHasta := utils.ParseDate(layout, req.FechaHasta)

	dailyRate := montoDia * (float64(porcentaje) / 100.0)
	total := dailyRate * dias

	viaticoDetalles := []models.DetalleViatico{
		{
			FechaDesde: fechaDesde,
			FechaHasta: fechaHasta,
			Dias:       dias,
			Lugar:      req.Lugar,
			MontoDia:   montoDia,
			Porcentaje: porcentaje,
			SubTotal:   total,
		},
	}

	rcivaPorcentajeStr := s.configService.GetValue(ctx, "IMPUESTO_RC_IVA")
	rcIvaRate := 0.13
	if rcivaPorcentajeStr != "" {
		val := utils.ParseFloat(rcivaPorcentajeStr)
		if val > 0 {
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
		UsuarioID:            userID,
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
		Codigo:               utils.GeneratePrefixedCode("V-", 8),
		Glosa:                "Asignación automática de viáticos",
	}

	if err := s.repo.WithContext(ctx).Create(viatico); err != nil {
		return nil, err
	}

	return viatico, nil
}

func (s *ViaticoService) GetByContext(ctx context.Context) ([]models.Viatico, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *ViaticoService) GetBySolicitud(ctx context.Context, solicitudID string) ([]models.Viatico, error) {
	return s.repo.WithContext(ctx).FindBySolicitudID(solicitudID)
}

func (s *ViaticoService) GetByID(ctx context.Context, id string) (*models.Viatico, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *ViaticoService) GetCategorias(ctx context.Context) ([]models.CategoriaViatico, error) {
	return s.catRepo.WithContext(ctx).FindAll()
}

func (s *ViaticoService) GetZonas(ctx context.Context) ([]models.ZonaViatico, error) {
	return s.zonaRepo.WithContext(ctx).FindAll()
}

func (s *ViaticoService) CreateZona(ctx context.Context, nombre string) error {
	zona := &models.ZonaViatico{
		Nombre: nombre,
	}
	return s.zonaRepo.WithContext(ctx).Create(zona)
}
