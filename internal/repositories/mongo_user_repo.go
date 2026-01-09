package repositories

import (
	"context"
	"errors"
	"sistema-pasajes/internal/configs"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MongoUser struct {
	ID       primitive.ObjectID `bson:"_id"`
	Username string             `bson:"username"`
	Password string             `bson:"password"`
	CI       string             `bson:"ci"`
	Roles    []string           `bson:"role_rrhh"`
}

type MongoUserRepository struct {
	ctx context.Context
}

func NewMongoUserRepository() *MongoUserRepository {
	return &MongoUserRepository{ctx: context.Background()}
}

func (r *MongoUserRepository) WithContext(ctx context.Context) *MongoUserRepository {
	return &MongoUserRepository{ctx: ctx}
}

func (r *MongoUserRepository) FindByUsername(username string) (*MongoUser, error) {
	if configs.MongoChat == nil {
		return nil, errors.New("conexión a MongoDB no establecida")
	}

	collection := configs.MongoChat.Collection("users")
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
	if configs.MongoChat == nil {
		return nil, errors.New("conexión a MongoDB no establecida")
	}

	collection := configs.MongoChat.Collection("users")
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
	err := collection.FindOne(ctx, bson.M{"ci": ci}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
