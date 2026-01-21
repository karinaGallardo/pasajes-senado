package services

import (
	"context"
	"errors"
	"strings"

	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo   *repositories.UsuarioRepository
	mongoUser  *repositories.MongoUserRepository
	peopleRepo *repositories.PeopleViewRepository
	rolRepo    *repositories.RolRepository
	generoRepo *repositories.GeneroRepository
}

func NewAuthService() *AuthService {
	return &AuthService{
		userRepo:   repositories.NewUsuarioRepository(),
		mongoUser:  repositories.NewMongoUserRepository(),
		peopleRepo: repositories.NewPeopleViewRepository(),
		rolRepo:    repositories.NewRolRepository(),
		generoRepo: repositories.NewGeneroRepository(),
	}
}

func (s *AuthService) Authenticate(ctx context.Context, username, password string) (*models.Usuario, error) {
	mongoData, err := s.verifyMongoCredentials(ctx, username, password)
	if err != nil {
		return nil, err
	}

	mongoUsername := utils.GetStringFromBson(mongoData, "username")
	mongoCI := utils.GetStringFromBson(mongoData, "ci")

	if mongoUsername != "" {
		user, err := s.userRepo.WithContext(ctx).FindByUsername(mongoUsername)
		if err == nil {
			return user, nil
		}
	} else if mongoCI != "" {
		user, err := s.userRepo.WithContext(ctx).FindByCI(mongoCI)
		if err == nil {
			return user, nil
		}
	}

	return nil, errors.New("el usuario no está registrado en el sistema local (contacte admin)")
}

func (s *AuthService) verifyMongoCredentials(ctx context.Context, username, password string) (bson.M, error) {
	result := bson.M{
		"_id":      "",
		"username": "",
		"ci":       "",
	}
	username = strings.ToLower(username)

	user, err := s.mongoUser.WithContext(ctx).FindByUsername(username)
	if err != nil {
		user, err = s.mongoUser.WithContext(ctx).FindByCI(username)
		if err != nil {
			return nil, errors.New("usuario no encontrado")
		}
		result["_id"] = user.ID
		result["ci"] = user.CI
	} else {
		result["_id"] = user.ID
		result["username"] = user.Username
	}

	storedPwd := user.Password
	if err := bcrypt.CompareHashAndPassword([]byte(storedPwd), []byte(password)); err != nil {
		return nil, errors.New("credenciales inválidas")
	}

	return result, nil
}
