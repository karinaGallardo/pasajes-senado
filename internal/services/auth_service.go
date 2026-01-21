package services

import (
	"context"
	"errors"
	"fmt"
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
	var localUser *models.Usuario
	var err error

	localUser, err = s.userRepo.WithContext(ctx).FindByUsername(strings.ToLower(username))
	if err != nil {
		localUser, err = s.userRepo.WithContext(ctx).FindByCI(username)
	}

	if localUser != nil && localUser.IsBlocked {
		return nil, errors.New("su cuenta ha sido bloqueada por demasiados intentos fallidos. Contacte a un administrador")
	}

	mongoData, err := s.verifyMongoCredentials(ctx, username, password)
	if err != nil {
		if err.Error() == "credenciales inválidas" && localUser != nil {
			localUser.LoginAttempts++
			remaining := 3 - localUser.LoginAttempts
			if localUser.LoginAttempts >= 3 {
				localUser.IsBlocked = true
				s.userRepo.WithContext(ctx).Update(localUser)
				return nil, errors.New("credenciales inválidas. Su cuenta ha sido bloqueada por demasiados intentos fallidos")
			}
			s.userRepo.WithContext(ctx).Update(localUser)
			return nil, fmt.Errorf("credenciales inválidas. Le quedan %d intentos antes de que su cuenta sea bloqueada", remaining)
		}
		return nil, err
	}

	mongoUsername := utils.GetStringFromBson(mongoData, "username")
	mongoCI := utils.GetStringFromBson(mongoData, "ci")

	var user *models.Usuario
	if mongoUsername != "" {
		user, err = s.userRepo.WithContext(ctx).FindByUsername(mongoUsername)
	} else if mongoCI != "" {
		user, err = s.userRepo.WithContext(ctx).FindByCI(mongoCI)
	}

	if err != nil || user == nil {
		return nil, errors.New("el usuario no está registrado en el sistema local (contacte admin)")
	}

	if user.LoginAttempts > 0 {
		user.LoginAttempts = 0
		user.IsBlocked = false
		s.userRepo.WithContext(ctx).Update(user)
	}

	return user, nil
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
