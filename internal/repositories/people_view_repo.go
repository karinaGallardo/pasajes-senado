package repositories

import (
	"context"
	"errors"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type PeopleViewRepository struct {
	db *mongo.Database
}

func NewPeopleViewRepository() *PeopleViewRepository {
	return &PeopleViewRepository{db: configs.MongoRRHH}
}

func (r *PeopleViewRepository) FindSenatorDataByCI(ci string) (*models.MongoPersonaView, error) {
	if r.db == nil {
		return nil, errors.New("conexión a MongoDB RRHH no establecida")
	}

	collection := r.db.Collection("view_people_pasajes")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result models.MongoPersonaView
	filter := bson.M{"ci": ci}

	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
