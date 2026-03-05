package repositories

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoUser struct {
	ID       primitive.ObjectID `bson:"_id"`
	Username string             `bson:"username"`
	Password string             `bson:"password"`
	CI       string             `bson:"ci"`
	Roles    []string           `bson:"role_rrhh"`
}

type MongoUserRepository struct {
	db  *mongo.Database
	ctx context.Context
}

func NewMongoUserRepository(db *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{
		db:  db,
		ctx: context.Background(),
	}
}

func (r *MongoUserRepository) WithContext(ctx context.Context) *MongoUserRepository {
	return &MongoUserRepository{
		db:  r.db,
		ctx: ctx,
	}
}

func (r *MongoUserRepository) FindByUsername(username string) (*MongoUser, error) {
	if r.db == nil {
		return nil, errors.New("conexión a MongoDB no establecida")
	}

	collection := r.db.Collection("users")
	var ctx context.Context
	var cancel context.CancelFunc

	if r.ctx != nil && r.ctx != context.Background() {
		ctx = r.ctx
		cancel = func() {}
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	}
	defer cancel()

	var user MongoUser
	err := collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *MongoUserRepository) FindByCI(ci string) (*MongoUser, error) {
	if r.db == nil {
		return nil, errors.New("conexión a MongoDB no establecida")
	}

	collection := r.db.Collection("users")
	var ctx context.Context
	var cancel context.CancelFunc

	if r.ctx != nil && r.ctx != context.Background() {
		ctx = r.ctx
		cancel = func() {}
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	}
	defer cancel()

	var user MongoUser
	opts := options.FindOne().SetSort(bson.D{{Key: "_id", Value: -1}})
	err := collection.FindOne(ctx, bson.M{"ci": ci}, opts).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
