package services

import (
	"context"
	"log/slog"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
)

type UsuarioService struct {
	repo          *repositories.UsuarioRepository
	peopleRepo    *repositories.PeopleViewRepository
	deptoRepo     *repositories.DepartamentoRepository
	mongoUserRepo *repositories.MongoUserRepository
	rolRepo       *repositories.RolRepository
	destinoRepo   *repositories.DestinoRepository
	cargoRepo     *repositories.CargoRepository
	oficinaRepo   *repositories.OficinaRepository
}

func NewUsuarioService(
	repo *repositories.UsuarioRepository,
	peopleRepo *repositories.PeopleViewRepository,
	deptoRepo *repositories.DepartamentoRepository,
	mongoUserRepo *repositories.MongoUserRepository,
	rolRepo *repositories.RolRepository,
	destinoRepo *repositories.DestinoRepository,
	cargoRepo *repositories.CargoRepository,
	oficinaRepo *repositories.OficinaRepository,
) *UsuarioService {
	return &UsuarioService{
		repo:          repo,
		peopleRepo:    peopleRepo,
		deptoRepo:     deptoRepo,
		mongoUserRepo: mongoUserRepo,
		rolRepo:       rolRepo,
		destinoRepo:   destinoRepo,
		cargoRepo:     cargoRepo,
		oficinaRepo:   oficinaRepo,
	}
}

func (s *UsuarioService) SyncStaff(ctx context.Context) (dtos.SyncResult, error) {
	mongoStaff, err := s.peopleRepo.WithContext(ctx).FindAllActiveStaff()
	if err != nil {
		return dtos.SyncResult{}, err
	}

	mongoMap := make(map[string]models.MongoPersonaView)
	for _, m := range mongoStaff {
		cleanCI := utils.CleanString(m.CI)
		if cleanCI != "" {
			mongoMap[cleanCI] = m
		}
	}

	pgUsers, _ := s.repo.FindByRoleType(ctx, models.RolFuncionario)
	for _, user := range pgUsers {
		if user.IsSenador() {
			continue
		}
		if _, exists := mongoMap[user.CI]; !exists {
			s.repo.Delete(ctx, &user)
		}
	}

	var result dtos.SyncResult
	for ci, mStaff := range mongoMap {
		user, err := s.repo.FindByCIUnscoped(ctx, ci)
		exists := err == nil

		// Obtener el username objetivo desde Mongo
		mongoUser, _ := s.mongoUserRepo.WithContext(ctx).FindByCI(ci)
		targetUsername := ci
		if mongoUser != nil && mongoUser.Username != "" {
			targetUsername = mongoUser.Username
		}

		// 3. Resolución de conflicto de Username (Prioridad absoluta al CI)
		checkUser, _ := s.repo.FindByUsernameUnscoped(ctx, targetUsername)
		if checkUser != nil && checkUser.CI != ci {
			// El nombre de usuario está tomado por OTRO CI.
			// Seguimos con el CI actual pero le marcamos el username como observado.
			slog.Warn("[Sync] Username colisionado con otro CI. Marcando como observado.",
				"username", targetUsername,
				"ci", ci,
				"ci_en_uso_por", checkUser.CI)

			targetUsername = targetUsername + "_observado"
		}

		if exists {
			if user.IsSenador() {
				continue
			}
			s.repo.Restore(ctx, user)
		} else {
			user = &models.Usuario{}
			user.CI = ci
			user.ID = mStaff.ID.Hex()
		}

		user.Username = targetUsername

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
			if depto, err := s.deptoRepo.FindByNombre(ctx, dept); err == nil {
				user.DepartamentoCode = &depto.Codigo
			}
		}

		cargoName := mStaff.Cargo
		if cargoName != "" {
			if cargo, err := s.cargoRepo.FindByDescripcion(ctx, cargoName); err == nil {
				user.CargoID = &cargo.ID
			} else {
				if nextCode, err := s.cargoRepo.GetNextCodigo(ctx); err == nil {
					newCargo := &models.Cargo{
						Codigo:      nextCode,
						Descripcion: cargoName,
						Categoria:   0,
					}
					if err := s.cargoRepo.Create(ctx, newCargo); err == nil {
						user.CargoID = &newCargo.ID
					}
				}
			}
		} else {
			user.CargoID = nil
		}

		oficinaName := mStaff.Dependencia
		if oficinaName != "" {
			if oficina, err := s.oficinaRepo.FindByDetalle(ctx, oficinaName); err == nil {
				user.OficinaID = &oficina.ID
			} else {
				if nextCode, err := s.oficinaRepo.GetNextCodigo(ctx); err == nil {
					newOficina := &models.Oficina{
						Codigo:  nextCode,
						Detalle: oficinaName,
					}
					if err := s.oficinaRepo.Create(ctx, newOficina); err == nil {
						user.OficinaID = &newOficina.ID
					}
				}
			}
		} else {
			user.OficinaID = nil
		}

		if user.RolCodigo == nil {
			rol := models.RolFuncionario
			user.RolCodigo = &rol
		}

		if err := s.repo.Save(ctx, user); err == nil {
			result.Count++
		}
	}

	return result, nil
}

func (s *UsuarioService) SyncSenators(ctx context.Context) (dtos.SyncResult, error) {
	mongoSenators, err := s.peopleRepo.WithContext(ctx).FindAllActiveSenators()
	if err != nil {
		return dtos.SyncResult{}, err
	}

	mongoMap := make(map[string]models.MongoPersonaView)
	for _, m := range mongoSenators {
		cleanCI := utils.CleanString(m.CI)
		if cleanCI != "" {
			mongoMap[cleanCI] = m
		}
	}

	var result dtos.SyncResult

	err = s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.UsuarioRepository) error {
		pgSenators, _ := repoTx.FindAllSenators(ctx)

		for _, user := range pgSenators {
			if _, exists := mongoMap[user.CI]; !exists {
				repoTx.Delete(ctx, &user)
			}
		}

		for ci, mSen := range mongoMap {
			user, err := repoTx.FindByCIUnscoped(ctx, ci)
			exists := err == nil

			// Obtener el username objetivo desde Mongo
			mongoUser, _ := s.mongoUserRepo.WithContext(ctx).FindByCI(ci)
			targetUsername := ci
			if mongoUser != nil && mongoUser.Username != "" {
				targetUsername = mongoUser.Username
			}

			// 3. Resolución de conflicto de Username (Prioridad absoluta al CI)
			checkUser, _ := repoTx.FindByUsernameUnscoped(ctx, targetUsername)
			if checkUser != nil && checkUser.CI != ci {
				// El nombre de usuario está tomado por OTRO CI.
				// Seguimos con el CI actual pero le marcamos el username como observado.
				slog.Warn("[SyncSenators] Username colisionado con otro CI. Marcando como observado.",
					"username", targetUsername,
					"ci", ci,
					"ci_en_uso_por", checkUser.CI)

				targetUsername = targetUsername + "_observado"
			}

			if exists {
				repoTx.Restore(ctx, user)
			} else {
				user = &models.Usuario{}
				user.CI = ci
				user.ID = mSen.ID.Hex()
			}

			user.Username = targetUsername

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
				if depto, err := s.deptoRepo.FindByNombre(ctx, dept); err == nil {
					user.DepartamentoCode = &depto.Codigo
				}
			}

			cargoName := mSen.Cargo
			if cargoName != "" {
				if cargo, err := s.cargoRepo.FindByDescripcion(ctx, cargoName); err == nil {
					user.CargoID = &cargo.ID
				} else {
					if nextCode, err := s.cargoRepo.GetNextCodigo(ctx); err == nil {
						newCargo := &models.Cargo{
							Codigo:      nextCode,
							Descripcion: cargoName,
							Categoria:   0,
						}
						if err := s.cargoRepo.Create(ctx, newCargo); err == nil {
							user.CargoID = &newCargo.ID
						}
					}
				}
			} else {
				user.CargoID = nil
			}

			oficinaName := mSen.Dependencia
			if oficinaName != "" {
				if oficina, err := s.oficinaRepo.FindByDetalle(ctx, oficinaName); err == nil {
					user.OficinaID = &oficina.ID
				} else {
					if nextCode, err := s.oficinaRepo.GetNextCodigo(ctx); err == nil {
						newOficina := &models.Oficina{
							Codigo:  nextCode,
							Detalle: oficinaName,
						}
						if err := s.oficinaRepo.Create(ctx, newOficina); err == nil {
							user.OficinaID = &newOficina.ID
						}
					}
				}
			} else {
				user.OficinaID = nil
			}

			if user.RolCodigo == nil {
				senadorRole := models.RolSenador
				user.RolCodigo = &senadorRole
			}

			if err := repoTx.Save(ctx, user); err == nil {
				result.Count++
			}
		}

		// Segunda pasada para relaciones
		for _, mSen := range mongoSenators {
			tipo := mSen.TipoFuncionario
			if tipo != "SENADOR_SUPLENTE" && tipo != "SENADOR_TITULAR" {
				continue
			}

			ci := utils.CleanString(mSen.CI)
			user, err := repoTx.FindByCI(ctx, ci)
			if err != nil {
				continue
			}

			switch tipo {
			case "SENADOR_SUPLENTE":
				titularCI := utils.CleanString(mSen.SenadorData.SuTitularCI)
				if titularCI != "" {
					titular, err := repoTx.FindByCI(ctx, titularCI)
					if err == nil {
						user.TitularID = &titular.ID
						repoTx.Save(ctx, user)
					}
				}
			case "SENADOR_TITULAR":
				suplenteCI := utils.CleanString(mSen.SenadorData.SuSuplenteCI)
				if suplenteCI != "" {
					suplente, err := repoTx.FindByCI(ctx, suplenteCI)
					if err == nil {
						suplente.TitularID = &user.ID
						repoTx.Save(ctx, suplente)
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		return dtos.SyncResult{}, err
	}

	return result, nil
}

func (s *UsuarioService) GetAll(ctx context.Context) ([]models.Usuario, error) {
	return s.repo.FindAll(ctx)
}

func (s *UsuarioService) GetByRoleType(ctx context.Context, roleType string) ([]models.Usuario, error) {
	return s.repo.FindByRoleType(ctx, roleType)
}

func (s *UsuarioService) GetPaginated(ctx context.Context, roleType string, page, limit int, searchTerm string) (*repositories.PaginatedUsers, error) {
	return s.repo.FindPaginated(ctx, roleType, page, limit, searchTerm)
}

func (s *UsuarioService) GetByID(ctx context.Context, id string) (*models.Usuario, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *UsuarioService) GetByIDs(ctx context.Context, ids []string) ([]models.Usuario, error) {
	return s.repo.FindByIDs(ctx, ids)
}

func (s *UsuarioService) UpdateRol(ctx context.Context, id string, rolCodigo string) error {
	return s.repo.UpdateRol(ctx, id, rolCodigo)
}

func (s *UsuarioService) Update(ctx context.Context, usuario *models.Usuario) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.UsuarioRepository) error {
		if err := repoTx.Update(ctx, usuario); err != nil {
			return err
		}

		if usuario.Tipo == "SENADOR_TITULAR" {
			suplente, err := repoTx.FindSuplenteByTitularID(ctx, usuario.ID)
			if err == nil && suplente != nil {
				if err := repoTx.Update(ctx, suplente); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// GetCommonCatalogs devuelve destinos y funcionarios para el caché global del frontend.
func (s *UsuarioService) GetCommonCatalogs(ctx context.Context) ([]models.Destino, []models.Usuario, error) {
	destinos, err := s.destinoRepo.FindAll(ctx)
	if err != nil {
		return nil, nil, err
	}

	funcionarios, err := s.repo.FindByRoleType(ctx, models.RolFuncionario)
	if err != nil {
		return nil, nil, err
	}

	return destinos, funcionarios, nil
}

func (s *UsuarioService) GetSenatorsByEncargado(ctx context.Context, encargadoID string) ([]models.Usuario, error) {
	return s.repo.FindByEncargadoID(ctx, encargadoID)
}

func (s *UsuarioService) GetSuplenteByTitularID(ctx context.Context, titularID string) (*models.Usuario, error) {
	return s.repo.FindSuplenteByTitularID(ctx, titularID)
}

func (s *UsuarioService) SyncOrigenesAlternativos(ctx context.Context, usuarioID string, origins []string) error {
	return s.repo.SyncOrigenesAlternativos(ctx, usuarioID, origins)
}

func (s *UsuarioService) SearchStaff(ctx context.Context, query string) ([]models.Usuario, error) {
	return s.repo.SearchStaff(ctx, query)
}

func (s *UsuarioService) GetEditContext(ctx context.Context, userID string, authUser *models.Usuario) (*dtos.UserEditContext, error) {
	usuario, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	roles, _ := s.rolRepo.FindAll(ctx)
	destinos, _ := s.destinoRepo.FindAll(ctx)
	funcionarios, _ := s.repo.FindByRoleType(ctx, models.RolFuncionario)
	cargos, _ := s.cargoRepo.FindAll(ctx)
	oficinas, _ := s.oficinaRepo.FindAll(ctx)

	perms := usuario.GetPermissions(authUser)
	permissions := map[string]bool{
		"CanChangeRol":    perms.CanChangeRol,
		"CanChangeOrigin": perms.CanChangeOrigin,
		"CanChangeStaff":  perms.CanChangeStaff,
		"CanManageRoutes": perms.CanManageRoutes,
		"CanEditContact":  perms.CanEditContact,
		"HasOrigin":       perms.HasOrigin,
		"HasCargo":        perms.HasCargo,
		"HasOficina":      perms.HasOficina,
		"HasRol":          perms.HasRol,
		"HasEmail":        perms.HasEmail,
		"HasPhone":        perms.HasPhone,
	}

	return &dtos.UserEditContext{
		Usuario:      usuario,
		Roles:        roles,
		Destinos:     destinos,
		Funcionarios: funcionarios,
		Cargos:       cargos,
		Oficinas:     oficinas,
		Permissions:  permissions,
	}, nil
}
