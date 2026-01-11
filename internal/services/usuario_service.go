package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gorm.io/gorm"
)

type UsuarioService struct {
	repo          *repositories.UsuarioRepository
	peopleRepo    *repositories.PeopleViewRepository
	deptoRepo     *repositories.DepartamentoRepository
	mongoUserRepo *repositories.MongoUserRepository
}

func NewUsuarioService() *UsuarioService {
	return &UsuarioService{
		repo:          repositories.NewUsuarioRepository(),
		peopleRepo:    repositories.NewPeopleViewRepository(),
		deptoRepo:     repositories.NewDepartamentoRepository(),
		mongoUserRepo: repositories.NewMongoUserRepository(),
	}
}

func (s *UsuarioService) SyncStaff(ctx context.Context) (int, error) {
	mongoStaff, err := s.peopleRepo.WithContext(ctx).FindAllActiveStaff()
	if err != nil {
		return 0, err
	}

	mongoMap := make(map[string]models.MongoPersonaView)
	for _, m := range mongoStaff {
		cleanCI := utils.CleanString(m.CI)
		if cleanCI != "" {
			mongoMap[cleanCI] = m
		}
	}

	pgUsers, _ := s.repo.WithContext(ctx).FindByRoleType("FUNCIONARIO")
	for _, user := range pgUsers {
		if user.IsSenador() {
			continue
		}
		if _, exists := mongoMap[user.CI]; !exists {
			s.repo.GetDB().Delete(&user)
		}
	}

	count := 0
	for ci, mStaff := range mongoMap {
		user, err := s.repo.WithContext(ctx).FindByCIUnscoped(ci)
		exists := err == nil

		if exists {
			if user.IsSenador() {
				continue
			}
			s.repo.GetDB().Model(user).Unscoped().Update("deleted_at", nil)
		} else {
			user = &models.Usuario{}
			user.CI = ci
			if oid, ok := mStaff.ID.(primitive.ObjectID); ok {
				user.ID = oid.Hex()
			}
		}

		mongoUser, _ := s.mongoUserRepo.WithContext(ctx).FindByCI(ci)
		if mongoUser != nil && mongoUser.Username != "" {
			user.Username = mongoUser.Username
		} else {
			user.Username = ci
		}

		user.Firstname = utils.CleanName(utils.GetString(mStaff.Firstname))
		user.Secondname = utils.CleanName(utils.GetString(mStaff.Secondname))
		user.Lastname = utils.CleanName(utils.GetString(mStaff.Lastname))
		user.Surname = utils.CleanName(utils.GetString(mStaff.Surname))
		user.Tipo = utils.GetString(mStaff.TipoFuncionario)
		user.Email = utils.CleanString(utils.GetString(mStaff.Email))
		user.Phone = utils.CleanString(utils.GetString(mStaff.Phone))
		user.Address = utils.CleanString(utils.GetString(mStaff.Address))

		dept := utils.GetString(mStaff.SenadorData.Departamento)
		if dept != "" {
			if depto, err := s.deptoRepo.WithContext(ctx).FindByNombre(dept); err == nil {
				user.DepartamentoCode = &depto.Codigo
			}
		}

		if user.RolCodigo == nil {
			rol := "FUNCIONARIO"
			user.RolCodigo = &rol
		}

		if err := s.repo.WithContext(ctx).Save(user); err == nil {
			count++
		}
	}

	return count, nil
}

func (s *UsuarioService) SyncSenators(ctx context.Context) (int, error) {
	mongoSenators, err := s.peopleRepo.WithContext(ctx).FindAllActiveSenators()
	if err != nil {
		return 0, err
	}

	mongoMap := make(map[string]models.MongoPersonaView)
	for _, m := range mongoSenators {
		cleanCI := utils.CleanString(m.CI)
		if cleanCI != "" {
			mongoMap[cleanCI] = m
		}
	}

	var pgSenators []models.Usuario
	s.repo.GetDB().Where("tipo IN ?", []string{"SENADOR_TITULAR", "SENADOR_SUPLENTE"}).Find(&pgSenators)

	for _, user := range pgSenators {
		if _, exists := mongoMap[user.CI]; !exists {
			s.repo.GetDB().Delete(&user)
		}
	}

	count := 0
	for ci, mSen := range mongoMap {
		user, err := s.repo.WithContext(ctx).FindByCIUnscoped(ci)
		exists := err == nil

		if exists {
			s.repo.GetDB().Model(user).Unscoped().Update("deleted_at", nil)
		} else {
			user = &models.Usuario{}
			user.CI = ci
			if oid, ok := mSen.ID.(primitive.ObjectID); ok {
				user.ID = oid.Hex()
			}
		}

		mongoUser, _ := s.mongoUserRepo.WithContext(ctx).FindByCI(ci)
		if mongoUser != nil && mongoUser.Username != "" {
			user.Username = mongoUser.Username
		} else {
			user.Username = ci
		}

		user.Firstname = utils.CleanName(utils.GetString(mSen.Firstname))
		user.Secondname = utils.CleanName(utils.GetString(mSen.Secondname))
		user.Lastname = utils.CleanName(utils.GetString(mSen.Lastname))
		user.Surname = utils.CleanName(utils.GetString(mSen.Surname))
		user.Tipo = utils.GetString(mSen.TipoFuncionario)
		user.Email = utils.CleanString(utils.GetString(mSen.Email))
		user.Phone = utils.CleanString(utils.GetString(mSen.Phone))
		user.Address = utils.CleanString(utils.GetString(mSen.Address))

		dept := utils.GetString(mSen.SenadorData.Departamento)
		if dept != "" {
			if depto, err := s.deptoRepo.WithContext(ctx).FindByNombre(dept); err == nil {
				user.DepartamentoCode = &depto.Codigo
			}
		}

		if user.RolCodigo == nil {
			senadorRole := "SENADOR"
			user.RolCodigo = &senadorRole
		}

		if err := s.repo.WithContext(ctx).Save(user); err == nil {
			count++
		}
	}

	for _, mSen := range mongoSenators {
		tipo := utils.GetString(mSen.TipoFuncionario)
		if tipo != "SENADOR_SUPLENTE" && tipo != "SENADOR_TITULAR" {
			continue
		}

		ci := utils.CleanString(mSen.CI)
		user, err := s.repo.WithContext(ctx).FindByCI(ci)
		if err != nil {
			continue
		}

		switch tipo {
		case "SENADOR_SUPLENTE":
			titularCI := utils.CleanString(mSen.SenadorData.Titular)
			if titularCI != "" {
				titular, err := s.repo.WithContext(ctx).FindByCI(titularCI)
				if err == nil {
					user.TitularID = &titular.ID
					user.EncargadoID = titular.EncargadoID
					s.repo.WithContext(ctx).Save(user)
				}
			}
		case "SENADOR_TITULAR":
			suplenteCI := utils.CleanString(mSen.SenadorData.Suplente)
			if suplenteCI != "" {
				suplente, err := s.repo.WithContext(ctx).FindByCI(suplenteCI)
				if err == nil {
					suplente.TitularID = &user.ID
					suplente.EncargadoID = user.EncargadoID
					s.repo.WithContext(ctx).Save(suplente)
				}
			}
		}
	}

	return count, nil
}

func (s *UsuarioService) GetAll(ctx context.Context) ([]models.Usuario, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *UsuarioService) GetByRoleType(ctx context.Context, roleType string) ([]models.Usuario, error) {
	return s.repo.WithContext(ctx).FindByRoleType(roleType)
}

func (s *UsuarioService) GetPaginated(ctx context.Context, roleType string, page, limit int, searchTerm string) (*repositories.PaginatedUsers, error) {
	return s.repo.WithContext(ctx).FindPaginated(roleType, page, limit, searchTerm)
}

func (s *UsuarioService) GetByID(ctx context.Context, id string) (*models.Usuario, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *UsuarioService) GetByIDs(ctx context.Context, ids []string) ([]models.Usuario, error) {
	return s.repo.WithContext(ctx).FindByIDs(ids)
}

func (s *UsuarioService) UpdateRol(ctx context.Context, id string, rolCodigo string) error {
	return s.repo.WithContext(ctx).UpdateRol(id, rolCodigo)
}

func (s *UsuarioService) Update(ctx context.Context, usuario *models.Usuario) error {
	return s.repo.WithContext(ctx).GetDB().Transaction(func(tx *gorm.DB) error {
		repoTx := s.repo.WithTx(tx)

		if err := repoTx.Update(usuario); err != nil {
			return err
		}

		if usuario.Tipo == "SENADOR_TITULAR" {
			suplente, err := repoTx.FindSuplenteByTitularID(usuario.ID)
			if err == nil && suplente != nil {
				suplente.EncargadoID = usuario.EncargadoID
				if err := repoTx.Update(suplente); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (s *UsuarioService) GetSenatorsByEncargado(ctx context.Context, encargadoID string) ([]models.Usuario, error) {
	return s.repo.WithContext(ctx).FindByEncargadoID(encargadoID)
}

func (s *UsuarioService) GetSuplenteByTitularID(ctx context.Context, titularID string) (*models.Usuario, error) {
	return s.repo.WithContext(ctx).FindSuplenteByTitularID(titularID)
}
