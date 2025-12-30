package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

)

type RolService struct {
	repo *repositories.RolRepository
}

func NewRolService() *RolService {
	return &RolService{
		repo: repositories.NewRolRepository(),
	}
}

func (s *RolService) GetAll() ([]models.Rol, error) {
	return s.repo.FindAll()
}
