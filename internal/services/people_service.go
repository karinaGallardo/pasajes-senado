package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type PeopleService struct {
	repo *repositories.PeopleViewRepository
}

func NewPeopleService() *PeopleService {
	return &PeopleService{
		repo: repositories.NewPeopleViewRepository(),
	}
}

func (s *PeopleService) FindSenatorDataByCI(ci string) (*models.MongoPersonaView, error) {
	return s.repo.FindSenatorDataByCI(ci)
}
