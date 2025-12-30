package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

	"gorm.io/gorm"
)

type UsuarioService struct {
	repo *repositories.UsuarioRepository
}

func NewUsuarioService(db *gorm.DB) *UsuarioService {
	return &UsuarioService{
		repo: repositories.NewUsuarioRepository(db),
	}
}

func (s *UsuarioService) GetAll() ([]models.Usuario, error) {
	return s.repo.FindAll()
}

func (s *UsuarioService) GetByRoleType(roleType string) ([]models.Usuario, error) {
	return s.repo.FindByRoleType(roleType)
}

func (s *UsuarioService) GetByID(id string) (*models.Usuario, error) {
	return s.repo.FindByID(id)
}

func (s *UsuarioService) UpdateRol(id string, rolCodigo string) error {
	return s.repo.UpdateRol(id, rolCodigo)
}

func (s *UsuarioService) Update(usuario *models.Usuario) error {
	return s.repo.Update(usuario)
}
