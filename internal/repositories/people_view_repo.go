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

func (r *PeopleViewRepository) FindAllActiveSenators() ([]models.MongoPersonaView, error) {
	if r.db == nil {
		return nil, errors.New("conexión a MongoDB RRHH no establecida")
	}

	collection := r.db.Collection("view_people_pasajes")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"senador_data.active": true,
		"tipo_funcionario":    bson.M{"$in": []string{"SENADOR_TITULAR", "SENADOR_SUPLENTE"}},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.MongoPersonaView
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *PeopleViewRepository) FindAllActiveStaff() ([]models.MongoPersonaView, error) {
	if r.db == nil {
		return nil, errors.New("conexión a MongoDB RRHH no establecida")
	}

	collection := r.db.Collection("view_people_pasajes")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{
		"tipo_funcionario":    bson.M{"$in": []string{"FUNCIONARIO_PERMANENTE", "FUNCIONARIO_EVENTUAL"}},
		"senador_data.active": bson.M{"$ne": true},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []models.MongoPersonaView
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
