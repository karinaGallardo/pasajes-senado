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
	db  *mongo.Database
	ctx context.Context
}

func NewPeopleViewRepository() *PeopleViewRepository {
	return &PeopleViewRepository{
		db:  configs.MongoRRHH,
		ctx: context.Background(),
	}
}

func (r *PeopleViewRepository) WithContext(ctx context.Context) *PeopleViewRepository {
	return &PeopleViewRepository{
		db:  r.db,
		ctx: ctx,
	}
}

func (r *PeopleViewRepository) FindSenatorDataByCI(ci string) (*models.MongoPersonaView, error) {
	if r.db == nil {
		return nil, errors.New("conexión a MongoDB RRHH no establecida")
	}

	collection := r.db.Collection("view_people_pasajes")
	var ctx context.Context
	var cancel context.CancelFunc

	if r.ctx != nil && r.ctx != context.Background() {
		ctx = r.ctx
		cancel = func() {}
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	}
	defer cancel()

	var results []models.MongoPersonaView
	filter := bson.M{"ci": ci}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$addFields", Value: bson.M{
			"cargo": bson.M{
				"$cond": bson.M{
					"if":   bson.M{"$eq": []any{"$tipo_funcionario", "FUNCIONARIO_PERMANENTE"}},
					"then": "$funcionario_permanente.item_data.descripcion",
					"else": bson.M{
						"$cond": bson.M{
							"if":   bson.M{"$in": []any{"$tipo_funcionario", []string{"SENADOR_TITULAR", "SENADOR_SUPLENTE"}}},
							"then": "$tipo_funcionario",
							"else": "$funcionario_eventual.cargo",
						},
					},
				},
			},
			"dependencia": bson.M{
				"$cond": bson.M{
					"if":   bson.M{"$eq": []any{"$tipo_funcionario", "FUNCIONARIO_PERMANENTE"}},
					"then": "$funcionario_permanente.item_data.unit",
					"else": bson.M{
						"$cond": bson.M{
							"if":   bson.M{"$in": []any{"$tipo_funcionario", []string{"SENADOR_TITULAR", "SENADOR_SUPLENTE"}}},
							"then": "CAMARA DE SENADORES",
							"else": "$funcionario_eventual.unit_data.name",
						},
					},
				},
			},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	if err := cursor.All(ctx, &results); err != nil || len(results) == 0 {
		return nil, err
	}

	return &results[0], nil
}

func (r *PeopleViewRepository) FindAllActiveSenators() ([]models.MongoPersonaView, error) {
	if r.db == nil {
		return nil, errors.New("conexión a MongoDB RRHH no establecida")
	}

	collection := r.db.Collection("view_people_pasajes")
	var ctx context.Context
	var cancel context.CancelFunc

	if r.ctx != nil && r.ctx != context.Background() {
		ctx = r.ctx
		cancel = func() {}
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	}
	defer cancel()

	filter := bson.M{
		"senador_data.active": true,
		"tipo_funcionario":    bson.M{"$in": []string{"SENADOR_TITULAR", "SENADOR_SUPLENTE"}},
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$addFields", Value: bson.M{
			"cargo":       "$tipo_funcionario",
			"dependencia": "CAMARA DE SENADORES",
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
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
	var ctx context.Context
	var cancel context.CancelFunc

	if r.ctx != nil && r.ctx != context.Background() {
		ctx = r.ctx
		cancel = func() {}
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	}
	defer cancel()

	filter := bson.M{
		"tipo_funcionario":    bson.M{"$in": []string{"FUNCIONARIO_PERMANENTE", "FUNCIONARIO_EVENTUAL"}},
		"senador_data.active": bson.M{"$ne": true},
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$addFields", Value: bson.M{
			"cargo": bson.M{
				"$cond": bson.M{
					"if":   bson.M{"$eq": []any{"$tipo_funcionario", "FUNCIONARIO_PERMANENTE"}},
					"then": "$funcionario_permanente.item_data.descripcion",
					"else": "$funcionario_eventual.cargo",
				},
			},
			"dependencia": bson.M{
				"$cond": bson.M{
					"if":   bson.M{"$eq": []any{"$tipo_funcionario", "FUNCIONARIO_PERMANENTE"}},
					"then": "$funcionario_permanente.item_data.unit",
					"else": "$funcionario_eventual.unit_data.name",
				},
			},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
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
