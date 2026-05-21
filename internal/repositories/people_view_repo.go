package repositories

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sistema-pasajes/internal/models"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

//go:embed scripts/crear_materialized_view_mongo3.6.js
var crearMaterializedViewScript []byte

type PeopleViewRepository struct {
	db  *mongo.Database
	ctx context.Context
}

func NewPeopleViewRepository(db *mongo.Database) *PeopleViewRepository {
	return &PeopleViewRepository{
		db:  db,
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

	var result models.MongoPersonaView
	filter := bson.M{"ci": ci}

	if err := collection.FindOne(ctx, filter).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
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
		"tipo_funcionario":    primitive.Regex{Pattern: "SENADOR_", Options: "i"},
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

func (r *PeopleViewRepository) SyncView(ctx context.Context) error {
	if r.db == nil {
		return errors.New("conexión a MongoDB RRHH no establecida")
	}

	mongoPort := viper.GetString("MONGO_PORT")
	if mongoPort == "" {
		mongoPort = "27017"
	}
	mongoHost := viper.GetString("MONGO_HOST")
	if mongoHost == "" {
		mongoHost = "127.0.0.1"
	}
	mongoUser := viper.GetString("MONGO_USER")
	mongoPassword := viper.GetString("MONGO_PASSWORD")
	mongoDB := viper.GetString("MONGO_RRHH_DB")
	if mongoDB == "" {
		mongoDB = "rrhh_db"
	}

	var mongoURI string
	if mongoUser != "" && mongoPassword != "" {
		mongoURI = fmt.Sprintf("mongodb://%s:%s@%s:%s/%s?authSource=admin",
			mongoUser, mongoPassword, mongoHost, mongoPort, mongoDB)
	} else {
		mongoURI = fmt.Sprintf("mongodb://%s:%s/%s", mongoHost, mongoPort, mongoDB)
	}

	// Crear un archivo temporal para escribir el script embebido
	tempFile, err := os.CreateTemp("", "crear_materialized_view_*.js")
	if err != nil {
		return fmt.Errorf("error al crear archivo temporal para script: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(crearMaterializedViewScript); err != nil {
		tempFile.Close()
		return fmt.Errorf("error al escribir script en archivo temporal: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("error al cerrar archivo temporal de script: %w", err)
	}

	cmd := exec.CommandContext(ctx, "mongosh", "--quiet", mongoURI, tempFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error al ejecutar mongosh: %w (salida: %s)", err, string(output))
	}

	// Recrear el índice en la vista materializada para asegurar búsquedas rápidas por CI (FindSenatorDataByCI)
	// Ya que el stage $out elimina la colección y destruye sus índices.
	_, err = r.db.Collection("view_people_pasajes").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "ci", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("error al recrear el índice de CI en la vista: %w", err)
	}

	return nil
}
