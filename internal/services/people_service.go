package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type PeopleService struct {
	repo *repositories.PeopleViewRepository
}

func NewPeopleService(repo *repositories.PeopleViewRepository) *PeopleService {
	return &PeopleService{
		repo: repo,
	}
}

func (s *PeopleService) GetSenatorDataByCI(ctx context.Context, ci string) (*models.MongoPersonaView, error) {
	return s.repo.WithContext(ctx).FindSenatorDataByCI(ci)
}
