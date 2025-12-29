package repositories

import (
	"context"
	"errors"
	"sistema-pasajes/internal/configs"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type SenadorData struct {
	Departamento string `bson:"departamento"`
	Sigla        string `bson:"sigla"`
	Tipo         string `bson:"tipo"`
	Suplente     string `bson:"suplente"`
	Gestion      string `bson:"gestion"`
	Active       bool   `bson:"active"`
}

type ItemData struct {
	Unit string `bson:"unit"`
}

type FuncionarioPermanente struct {
	ItemData ItemData `bson:"item_data"`
}

type ItemUnitData struct {
	Name string `bson:"name"`
}

type FuncionarioEventual struct {
	UnitData ItemUnitData `bson:"unit_data"`
}

type MongoPersonaView struct {
	CI                    string                `bson:"ci"`
	SenadorData           SenadorData           `bson:"senador_data"`
	FuncionarioPermanente FuncionarioPermanente `bson:"funcionario_permanente"`
	FuncionarioEventual   FuncionarioEventual   `bson:"funcionario_eventual"`
}

type PeopleViewRepository struct{}

func NewPeopleViewRepository() *PeopleViewRepository {
	return &PeopleViewRepository{}
}

func (r *PeopleViewRepository) FindSenatorDataByCI(ci string) (*MongoPersonaView, error) {
	if configs.MongoRRHH == nil {
		return nil, errors.New("conexión a MongoDB RRHH no establecida")
	}

	collection := configs.MongoRRHH.Collection("view_people_pasajes")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result MongoPersonaView
	filter := bson.M{"ci": ci}

	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
