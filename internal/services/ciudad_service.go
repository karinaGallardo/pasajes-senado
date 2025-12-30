package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

	"gorm.io/gorm"
)

type CiudadService struct {
	repo *repositories.CiudadRepository
}

func NewCiudadService(db *gorm.DB) *CiudadService {
	return &CiudadService{
		repo: repositories.NewCiudadRepository(db),
	}
}

func (s *CiudadService) GetAll() ([]models.Ciudad, error) {
	return s.repo.FindAll()
}
