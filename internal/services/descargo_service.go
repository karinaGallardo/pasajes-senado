package services

import (
	"context"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
)

type DescargoService struct {
	repo *repositories.DescargoRepository
}

func NewDescargoService() *DescargoService {
	return &DescargoService{
		repo: repositories.NewDescargoRepository(),
	}
}

func (s *DescargoService) Create(ctx context.Context, req dtos.CreateDescargoRequest, userID string) (*models.Descargo, error) {
	fechaPresentacion := utils.ParseDate("2006-01-02", req.FechaPresentacion)
	monto := utils.ParseFloat(req.MontoDevolucion)
	codigo := utils.GenerateYearlyCode("D", 6)

	descargo := &models.Descargo{
		SolicitudID:        req.SolicitudID,
		UsuarioID:          userID,
		Codigo:             codigo,
		NumeroCite:         req.NumeroCite,
		FechaPresentacion:  fechaPresentacion,
		InformeActividades: req.InformeActividades,
		MontoDevolucion:    monto,
		Observaciones:      req.Observaciones,
		Estado:             "EN_REVISION",
	}
	descargo.CreatedBy = &userID

	var docs []models.DocumentoDescargo
	for i := range req.DocTipo {
		if i < len(req.DocNumero) && req.DocNumero[i] != "" {
			f := utils.ParseDate("2006-01-02", req.DocFecha[i])
			docs = append(docs, models.DocumentoDescargo{
				Tipo:    req.DocTipo[i],
				Numero:  req.DocNumero[i],
				Fecha:   f,
				Detalle: req.DocDetalle[i],
			})
		}
	}
	descargo.Documentos = docs

	if err := s.repo.WithContext(ctx).Create(descargo); err != nil {
		return nil, err
	}

	return descargo, nil
}

func (s *DescargoService) GetBySolicitudID(ctx context.Context, solicitudID string) (*models.Descargo, error) {
	return s.repo.WithContext(ctx).FindBySolicitudID(solicitudID)
}

func (s *DescargoService) GetByID(ctx context.Context, id string) (*models.Descargo, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *DescargoService) GetAll(ctx context.Context) ([]models.Descargo, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *DescargoService) Update(ctx context.Context, descargo *models.Descargo) error {
	return s.repo.WithContext(ctx).Update(descargo)
}
