package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

	"gorm.io/gorm"
)

type ConfiguracionService struct {
	repo *repositories.ConfiguracionRepository
}

func NewConfiguracionService(db *gorm.DB) *ConfiguracionService {
	return &ConfiguracionService{
		repo: repositories.NewConfiguracionRepository(db),
	}
}

func (s *ConfiguracionService) GetAll() ([]models.Configuracion, error) {
	return s.repo.FindAll()
}

func (s *ConfiguracionService) Update(clave, valor string) error {
	conf, err := s.repo.FindByClave(clave)
	if err != nil {
		return err
	}
	conf.Valor = valor
	return s.repo.Save(conf)
}

func (s *ConfiguracionService) GetValue(clave string) string {
	conf, err := s.repo.FindByClave(clave)
	if err != nil {
		return ""
	}
	return conf.Valor
}
