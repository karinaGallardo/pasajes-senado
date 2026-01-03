package services

import (
	"errors"
	"log"

	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func (s *AuthService) getOrInitUser(username string, authUser, profile bson.M) (*models.Usuario, error) {
	user, err := s.userRepo.FindByUsername(username)

	if err != nil {
		user = &models.Usuario{}
		user.Username = username

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
	}
	return user, nil
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

	genero, err := s.generoRepo.FirstOrCreate(genderName, genderName)
	if err != nil {
		return err
	}
	user.GeneroCodigo = &genero.Codigo
	return nil
}

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
	user, err := s.mongoUser.FindByUsername(username)
	if err != nil {
		return nil, errors.New("usuario no encontrado")
	}

	storedPwd := user.Password
	if err := bcrypt.CompareHashAndPassword([]byte(storedPwd), []byte(password)); err != nil {
		return nil, errors.New("credenciales inválidas")
	}

	result := bson.M{
		"_id":      user.ID,
		"username": user.Username,
		"ci":       user.CI,
		// "password": user.Password,
		// "role_rrhh": user.Roles,
	}
	return result, nil
}

func (s *AuthService) fetchUserProfile(ci string) (bson.M, error) {
	persona, err := s.peopleRepo.FindSenatorDataByCI(ci)
	if err != nil {
		return nil, err
	}
	if persona == nil {
		return nil, errors.New("perfil no encontrado")
	}

	result := bson.M{
		"_id":              persona.ID,
		"ci":               persona.CI,
		"firstname":        persona.Firstname,
		"secondname":       persona.Secondname,
		"lastname":         persona.Lastname,
		"surname":          persona.Surname,
		"phone":            persona.Phone,
		"address":          persona.Address,
		"email":            persona.Email,
		"gender":           persona.Gender,
		"tipo_funcionario": persona.TipoFuncionario,
	}

	return result, nil
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

	if err := s.userRepo.Save(user); err != nil {
		return nil, err
	}

	if user.Rol == nil && user.RolCodigo != nil {
		s.userRepo.Refresh(user)
	}
	return user, nil
}

func (s *AuthService) ensureDefaultRole(user *models.Usuario, profile bson.M) error {
	if user.RolCodigo != nil {
		return nil
	}

	targetRole := "FUNCIONARIO"

	if tipo := getString(profile, "tipo_funcionario"); tipo == "SENADOR_TITULAR" || tipo == "SENADOR_SUPLENTE" {
		targetRole = "SENADOR"
	}

	rol, err := s.rolRepo.FindByCodigo(targetRole)
	if err != nil {
		return nil
	}
	user.RolCodigo = &rol.Codigo
	user.Rol = rol
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
