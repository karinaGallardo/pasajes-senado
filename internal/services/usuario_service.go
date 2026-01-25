package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
)

type UsuarioService struct {
	repo          *repositories.UsuarioRepository
	peopleRepo    *repositories.PeopleViewRepository
	deptoRepo     *repositories.DepartamentoRepository
	mongoUserRepo *repositories.MongoUserRepository
	cargoRepo     *repositories.CargoRepository
	oficinaRepo   *repositories.OficinaRepository
}

func NewUsuarioService() *UsuarioService {
	return &UsuarioService{
		repo:          repositories.NewUsuarioRepository(),
		peopleRepo:    repositories.NewPeopleViewRepository(),
		deptoRepo:     repositories.NewDepartamentoRepository(),
		mongoUserRepo: repositories.NewMongoUserRepository(),
		cargoRepo:     repositories.NewCargoRepository(),
		oficinaRepo:   repositories.NewOficinaRepository(),
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
			s.repo.Delete(&user)
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
			s.repo.Restore(user)
		} else {
			user = &models.Usuario{}
			user.CI = ci
			user.ID = mStaff.ID.Hex()
		}

		mongoUser, _ := s.mongoUserRepo.WithContext(ctx).FindByCI(ci)
		if mongoUser != nil && mongoUser.Username != "" {
			user.Username = mongoUser.Username
		} else {
			user.Username = ci
		}

		user.Firstname = utils.CleanName(mStaff.Firstname)
		user.Secondname = utils.CleanName(mStaff.Secondname)
		user.Lastname = utils.CleanName(mStaff.Lastname)
		user.Surname = utils.CleanName(mStaff.Surname)
		user.Tipo = mStaff.TipoFuncionario

		if !exists {
			user.Email = utils.CleanString(mStaff.Email)
			user.Phone = utils.CleanString(mStaff.Phone)
		}

		user.Address = utils.CleanString(mStaff.Address)

		dept := mStaff.SenadorData.Departamento
		if dept != "" {
			if depto, err := s.deptoRepo.WithContext(ctx).FindByNombre(dept); err == nil {
				user.DepartamentoCode = &depto.Codigo
			}
		}

		cargoName := mStaff.Cargo
		if cargoName != "" {
			if cargo, err := s.cargoRepo.WithContext(ctx).FindByDescripcion(cargoName); err == nil {
				user.CargoID = &cargo.ID
			} else {
				if nextCode, err := s.cargoRepo.WithContext(ctx).GetNextCodigo(); err == nil {
					newCargo := &models.Cargo{
						Codigo:      nextCode,
						Descripcion: cargoName,
						Categoria:   0,
					}
					if err := s.cargoRepo.WithContext(ctx).Create(newCargo); err == nil {
						user.CargoID = &newCargo.ID
					}
				}
			}
		} else {
			user.CargoID = nil
		}

		oficinaName := mStaff.Dependencia
		if oficinaName != "" {
			if oficina, err := s.oficinaRepo.WithContext(ctx).FindByDetalle(oficinaName); err == nil {
				user.OficinaID = &oficina.ID
			} else {
				if nextCode, err := s.oficinaRepo.WithContext(ctx).GetNextCodigo(); err == nil {
					newOficina := &models.Oficina{
						Codigo:  nextCode,
						Detalle: oficinaName,
					}
					if err := s.oficinaRepo.WithContext(ctx).Create(newOficina); err == nil {
						user.OficinaID = &newOficina.ID
					}
				}
			}
		} else {
			user.OficinaID = nil
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

	var count int

	err = s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.UsuarioRepository) error {
		pgSenators, _ := repoTx.FindAllSenators()

		for _, user := range pgSenators {
			if _, exists := mongoMap[user.CI]; !exists {
				repoTx.Delete(&user)
			}
		}

		for ci, mSen := range mongoMap {
			user, err := repoTx.FindByCIUnscoped(ci)
			exists := err == nil

			if exists {
				repoTx.Restore(user)
			} else {
				user = &models.Usuario{}
				user.CI = ci
				user.ID = mSen.ID.Hex()
			}

			mongoUser, _ := s.mongoUserRepo.WithContext(ctx).FindByCI(ci)
			if mongoUser != nil && mongoUser.Username != "" {
				user.Username = mongoUser.Username
			} else {
				user.Username = ci
			}

			user.Firstname = utils.CleanName(mSen.Firstname)
			user.Secondname = utils.CleanName(mSen.Secondname)
			user.Lastname = utils.CleanName(mSen.Lastname)
			user.Surname = utils.CleanName(mSen.Surname)
			user.Tipo = mSen.TipoFuncionario

			if !exists {
				user.Email = utils.CleanString(mSen.Email)
				user.Phone = utils.CleanString(mSen.Phone)
			}

			user.Address = utils.CleanString(mSen.Address)

			dept := mSen.SenadorData.Departamento
			if dept != "" {
				if depto, err := s.deptoRepo.WithContext(ctx).FindByNombre(dept); err == nil {
					user.DepartamentoCode = &depto.Codigo
				}
			}

			cargoName := mSen.Cargo
			if cargoName != "" {
				if cargo, err := s.cargoRepo.WithContext(ctx).FindByDescripcion(cargoName); err == nil {
					user.CargoID = &cargo.ID
				} else {
					if nextCode, err := s.cargoRepo.WithContext(ctx).GetNextCodigo(); err == nil {
						newCargo := &models.Cargo{
							Codigo:      nextCode,
							Descripcion: cargoName,
							Categoria:   0,
						}
						if err := s.cargoRepo.WithContext(ctx).Create(newCargo); err == nil {
							user.CargoID = &newCargo.ID
						}
					}
				}
			} else {
				user.CargoID = nil
			}

			oficinaName := mSen.Dependencia
			if oficinaName != "" {
				if oficina, err := s.oficinaRepo.WithContext(ctx).FindByDetalle(oficinaName); err == nil {
					user.OficinaID = &oficina.ID
				} else {
					if nextCode, err := s.oficinaRepo.WithContext(ctx).GetNextCodigo(); err == nil {
						newOficina := &models.Oficina{
							Codigo:  nextCode,
							Detalle: oficinaName,
						}
						if err := s.oficinaRepo.WithContext(ctx).Create(newOficina); err == nil {
							user.OficinaID = &newOficina.ID
						}
					}
				}
			} else {
				user.OficinaID = nil
			}

			if user.RolCodigo == nil {
				senadorRole := "SENADOR"
				user.RolCodigo = &senadorRole
			}

			if err := repoTx.Save(user); err == nil {
				count++
			}
		}

		// Segunda pasada para relaciones
		for _, mSen := range mongoSenators {
			tipo := mSen.TipoFuncionario
			if tipo != "SENADOR_SUPLENTE" && tipo != "SENADOR_TITULAR" {
				continue
			}

			ci := utils.CleanString(mSen.CI)
			user, err := repoTx.FindByCI(ci)
			if err != nil {
				continue
			}

			switch tipo {
			case "SENADOR_SUPLENTE":
				titularCI := utils.CleanString(mSen.SenadorData.SuTitularCI)
				if titularCI != "" {
					titular, err := repoTx.FindByCI(titularCI)
					if err == nil {
						user.TitularID = &titular.ID
						user.EncargadoID = titular.EncargadoID
						repoTx.Save(user)
					}
				}
			case "SENADOR_TITULAR":
				suplenteCI := utils.CleanString(mSen.SenadorData.SuSuplenteCI)
				if suplenteCI != "" {
					suplente, err := repoTx.FindByCI(suplenteCI)
					if err == nil {
						suplente.TitularID = &user.ID
						suplente.EncargadoID = user.EncargadoID
						repoTx.Save(suplente)
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return 0, err
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
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.UsuarioRepository) error {
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
