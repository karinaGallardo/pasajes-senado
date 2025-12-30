package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

)

type AgenciaService struct {
	repo *repositories.AgenciaRepository
}

func NewAgenciaService() *AgenciaService {
	return &AgenciaService{
		repo: repositories.NewAgenciaRepository(),
	}
}

func (s *AgenciaService) GetAllActive() ([]models.Agencia, error) {
	return s.repo.FindAllActive()
}

func (s *AgenciaService) GetAll() ([]models.Agencia, error) {
	return s.repo.FindAll()
}

func (s *AgenciaService) Create(nombre, telefono string) (*models.Agencia, error) {
	agencia := &models.Agencia{Nombre: nombre, Telefono: telefono, Estado: true}
	err := s.repo.Create(agencia)
	return agencia, err
}

func (s *AgenciaService) Toggle(id string) error {
	a, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	a.Estado = !a.Estado
	return s.repo.Save(a)
}
