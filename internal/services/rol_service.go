package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

	"gorm.io/gorm"
)

type RolService struct {
	repo *repositories.RolRepository
}

func NewRolService(db *gorm.DB) *RolService {
	return &RolService{
		repo: repositories.NewRolRepository(db),
	}
}

func (s *RolService) GetAll() ([]models.Rol, error) {
	return s.repo.FindAll()
}
