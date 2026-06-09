package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type ConfiguracionService struct {
	repo *repositories.ConfiguracionRepository
}

func NewConfiguracionService(repo *repositories.ConfiguracionRepository) *ConfiguracionService {
	return &ConfiguracionService{
		repo: repo,
	}
}

func (s *ConfiguracionService) GetAll(ctx context.Context) ([]models.Configuracion, error) {
	return s.repo.FindAll(ctx)
}

func (s *ConfiguracionService) Update(ctx context.Context, config *models.Configuracion) error {
	existing, err := s.repo.FindByClave(ctx, config.Clave)
	if err != nil {
		return err
	}
	existing.Valor = config.Valor
	return s.repo.Update(ctx, existing)
}

func (s *ConfiguracionService) GetValue(ctx context.Context, clave string) string {
	conf, err := s.repo.FindByClave(ctx, clave)
	if err != nil {
		return ""
	}
	return conf.Valor
}

func (s *ConfiguracionService) GetBankDefaults(ctx context.Context) (cuenta, nombre string) {
	cuenta = s.GetValue(ctx, "BANCO_CUENTA_DEVOLUCION")
	if cuenta == "" {
		cuenta = "10000005588211"
	}
	nombre = s.GetValue(ctx, "BANCO_NOMBRE_DEVOLUCION")
	if nombre == "" {
		nombre = "BANCO UNIÓN S.A."
	}
	return
}
