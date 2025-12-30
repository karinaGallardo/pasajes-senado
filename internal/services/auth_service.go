package services

import (
	"context"
	"errors"
	"log"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/utils"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

type AuthService struct {
	db        *gorm.DB
	mongoChat *mongo.Database
	mongoRRHH *mongo.Database
}

func NewAuthService(db *gorm.DB, mongoChat *mongo.Database, mongoRRHH *mongo.Database) *AuthService {
	return &AuthService{
		db:        db,
		mongoChat: mongoChat,
		mongoRRHH: mongoRRHH,
	}
}

func (s *AuthService) AuthenticateAndSync(username, password string) (*models.Usuario, error) {
	mongoAuthUser, err := s.verifyMongoCredentials(username, password)
	if err != nil {
		return nil, err
	}
	profileCI := getString(mongoAuthUser, "ci")
	if profileCI == "" {
		profileCI = username
	}

	userProfile, err := s.fetchUserProfile(profileCI)
	if err != nil {
		log.Printf("Warn: No perfil extendido para CI %s. Usando básicos.", profileCI)
		userProfile = mongoAuthUser
	}

	return s.syncUserToPostgres(mongoAuthUser, userProfile, username)
}

func (s *AuthService) verifyMongoCredentials(username, password string) (bson.M, error) {
	if s.mongoChat == nil {
		return nil, errors.New("MongoDB Chat not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mongoUser bson.M
	err := s.mongoChat.Collection("users").FindOne(ctx, bson.M{"username": username}).Decode(&mongoUser)
	if err != nil {
		return nil, errors.New("usuario no encontrado")
	}

	/*
		storedPwd, _ := mongoUser["password"].(string)
		if err := bcrypt.CompareHashAndPassword([]byte(storedPwd), []byte(password)); err != nil {
			return nil, errors.New("credenciales inválidas")
		}
	*/

	return mongoUser, nil
}

func (s *AuthService) fetchUserProfile(ci string) (bson.M, error) {
	if s.mongoRRHH == nil {
		return nil, errors.New("MongoDB RRHH not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var profile bson.M
	err := s.mongoRRHH.Collection("view_people_pasajes").FindOne(ctx, bson.M{"ci": ci}).Decode(&profile)
	return profile, err
}

func (s *AuthService) syncUserToPostgres(authUser, profile bson.M, username string) (*models.Usuario, error) {
	user, err := s.getOrInitUser(username, authUser, profile)
	if err != nil {
		return nil, err
	}

	s.mapProfileToUser(user, profile)

	if err := s.resolveGender(user, profile); err != nil {
		return nil, err
	}
	if err := s.ensureDefaultRole(user, profile); err != nil {
		return nil, err
	}

	if err := s.db.Save(user).Error; err != nil {
		return nil, err
	}
	if user.Rol == nil && user.RolID != nil {
		s.db.Preload("Rol").First(user)
	}

	return user, nil
}

func (s *AuthService) getOrInitUser(username string, authUser, profile bson.M) (*models.Usuario, error) {
	var user models.Usuario
	err := s.db.Preload("Rol").Where("username = ?", username).First(&user).Error

	if err != nil {
		idSet := false
		if profile != nil {
			if idObj, ok := profile["_id"].(primitive.ObjectID); ok {
				user.ID = idObj.Hex()
				idSet = true
			}
		}
		if !idSet {
			if idObj, ok := authUser["_id"].(primitive.ObjectID); ok {
				user.ID = idObj.Hex()
			}
		}
		user.Username = username
	}
	return &user, nil
}

func (s *AuthService) mapProfileToUser(user *models.Usuario, profile bson.M) {
	user.Firstname = utils.CleanName(getString(profile, "firstname"))
	user.Secondname = utils.CleanName(getString(profile, "secondname"))
	user.Lastname = utils.CleanName(getString(profile, "lastname"))
	user.Surname = utils.CleanName(getString(profile, "surname"))
	user.CI = utils.CleanString(getString(profile, "ci"))
	user.Phone = utils.CleanString(getString(profile, "phone"))
	user.Address = utils.CleanString(getString(profile, "address"))
	user.Tipo = utils.CleanString(getString(profile, "tipo_funcionario"))

	if email := getString(profile, "email"); email != "" {
		user.Email = utils.CleanString(email)
	}
}

func (s *AuthService) resolveGender(user *models.Usuario, profile bson.M) error {
	genderName := utils.CleanString(getString(profile, "gender"))
	if genderName == "" {
		return nil
	}

	var genero models.Genero
	if err := s.db.FirstOrCreate(&genero, models.Genero{Codigo: genderName, Nombre: genderName}).Error; err != nil {
		return err
	}
	user.GeneroID = &genero.Codigo
	return nil
}

func (s *AuthService) ensureDefaultRole(user *models.Usuario, profile bson.M) error {
	if user.RolID != nil {
		return nil
	}

	targetRole := "FUNCIONARIO"

	if tipo := getString(profile, "tipo_funcionario"); tipo == "SENADOR_TITULAR" || tipo == "SENADOR_SUPLENTE" {
		targetRole = "SENADOR"
	}

	var rol models.Rol
	if err := s.db.Where("codigo = ?", targetRole).First(&rol).Error; err != nil {
		return nil
	}
	user.RolID = &rol.Codigo
	user.Rol = &rol
	return nil
}

func getString(m bson.M, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
