package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UsuarioService struct {
	repo       *repositories.UsuarioRepository
	peopleRepo *repositories.PeopleViewRepository
	ciudadRepo *repositories.CiudadRepository
	deptoRepo  *repositories.DepartamentoRepository
}

func NewUsuarioService() *UsuarioService {
	return &UsuarioService{
		repo:       repositories.NewUsuarioRepository(),
		peopleRepo: repositories.NewPeopleViewRepository(),
		ciudadRepo: repositories.NewCiudadRepository(),
		deptoRepo:  repositories.NewDepartamentoRepository(),
	}
}

func (s *UsuarioService) SyncStaff() (int, error) {
	mongoStaff, err := s.peopleRepo.FindAllActiveStaff()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, mStaff := range mongoStaff {
		ci := utils.CleanString(mStaff.CI)
		if ci == "" {
			continue
		}

		user, err := s.repo.FindByCI(ci)
		exists := err == nil

		if exists && (user.Tipo == "SENADOR_TITULAR" || user.Tipo == "SENADOR_SUPLENTE") {
			continue
		}

		if !exists {
			user = &models.Usuario{}
			user.CI = ci
			user.Username = ci
			if oid, ok := mStaff.ID.(primitive.ObjectID); ok {
				user.ID = oid.Hex()
			}
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
			if depto, err := s.deptoRepo.FindByNombre(dept); err == nil {
				user.DepartamentoCode = &depto.Codigo
			}
		}

		if user.RolCodigo == nil {
			rol := "FUNCIONARIO"
			user.RolCodigo = &rol
		}

		if err := s.repo.Save(user); err == nil {
			count++
		}
	}

	return count, nil
}

func (s *UsuarioService) SyncSenators() (int, error) {
	mongoSenators, err := s.peopleRepo.FindAllActiveSenators()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, mSen := range mongoSenators {
		ci := utils.CleanString(mSen.CI)
		if ci == "" {
			continue
		}

		user, err := s.repo.FindByCI(ci)
		exists := err == nil

		if !exists {
			user = &models.Usuario{}
			user.CI = ci
			user.Username = ci
			if oid, ok := mSen.ID.(primitive.ObjectID); ok {
				user.ID = oid.Hex()
			}
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
			if depto, err := s.deptoRepo.FindByNombre(dept); err == nil {
				user.DepartamentoCode = &depto.Codigo
			}
		}

		if user.RolCodigo == nil {
			senadorRole := "SENADOR"
			user.RolCodigo = &senadorRole
		}

		if err := s.repo.Save(user); err == nil {
			count++
		}
	}

	for _, mSen := range mongoSenators {
		tipo := utils.GetString(mSen.TipoFuncionario)
		if tipo != "SENADOR_SUPLENTE" && tipo != "SENADOR_TITULAR" {
			continue
		}

		ci := utils.CleanString(mSen.CI)
		user, err := s.repo.FindByCI(ci)
		if err != nil {
			continue
		}

		if tipo == "SENADOR_SUPLENTE" {
			titularCI := utils.CleanString(mSen.SenadorData.Titular)
			if titularCI != "" {
				titular, err := s.repo.FindByCI(titularCI)
				if err == nil {
					user.TitularID = &titular.ID
					user.EncargadoID = titular.EncargadoID
					s.repo.Save(user)
				}
			}
		} else if tipo == "SENADOR_TITULAR" {
			suplenteCI := utils.CleanString(mSen.SenadorData.Suplente)
			if suplenteCI != "" {
				suplente, err := s.repo.FindByCI(suplenteCI)
				if err == nil {
					suplente.TitularID = &user.ID
					suplente.EncargadoID = user.EncargadoID
					s.repo.Save(suplente)
				}
			}
		}
	}

	return count, nil
}

func (s *UsuarioService) GetAll() ([]models.Usuario, error) {
	return s.repo.FindAll()
}

func (s *UsuarioService) GetByRoleType(roleType string) ([]models.Usuario, error) {
	return s.repo.FindByRoleType(roleType)
}

func (s *UsuarioService) GetPaginated(roleType string, page, limit int, searchTerm string) (*repositories.PaginatedUsers, error) {
	return s.repo.FindPaginated(roleType, page, limit, searchTerm)
}

func (s *UsuarioService) GetByID(id string) (*models.Usuario, error) {
	return s.repo.FindByID(id)
}

func (s *UsuarioService) UpdateRol(id string, rolCodigo string) error {
	return s.repo.UpdateRol(id, rolCodigo)
}

func (s *UsuarioService) Update(usuario *models.Usuario) error {
	err := s.repo.Update(usuario)
	if err != nil {
		return err
	}

	if usuario.Tipo == "SENADOR_TITULAR" {
		suplente, err := s.repo.FindSuplenteByTitularID(usuario.ID)
		if err == nil && suplente != nil {
			suplente.EncargadoID = usuario.EncargadoID
			return s.repo.Update(suplente)
		}
	}

	return nil
}

func (s *UsuarioService) GetSenatorsByEncargado(encargadoID string) ([]models.Usuario, error) {
	return s.repo.FindByEncargadoID(encargadoID)
}

func (s *UsuarioService) GetSuplenteByTitularID(titularID string) (*models.Usuario, error) {
	return s.repo.FindSuplenteByTitularID(titularID)
}
