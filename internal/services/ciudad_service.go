package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

)

type CiudadService struct {
	repo *repositories.CiudadRepository
}

func NewCiudadService() *CiudadService {
	return &CiudadService{
		repo: repositories.NewCiudadRepository(),
	}
}

func (s *CiudadService) GetAll() ([]models.Ciudad, error) {
	return s.repo.FindAll()
}
