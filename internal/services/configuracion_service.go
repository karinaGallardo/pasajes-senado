package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type ConfiguracionService struct {
	repo *repositories.ConfiguracionRepository
}

func NewConfiguracionService() *ConfiguracionService {
	return &ConfiguracionService{
		repo: repositories.NewConfiguracionRepository(),
	}
}

func (s *ConfiguracionService) GetAll(ctx context.Context) ([]models.Configuracion, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *ConfiguracionService) Update(ctx context.Context, clave, valor string) error {
	conf, err := s.repo.WithContext(ctx).FindByClave(clave)
	if err != nil {
		return err
	}
	conf.Valor = valor
	return s.repo.WithContext(ctx).Save(conf)
}

func (s *ConfiguracionService) GetValue(ctx context.Context, clave string) string {
	conf, err := s.repo.WithContext(ctx).FindByClave(clave)
	if err != nil {
		return ""
	}
	return conf.Valor
}
