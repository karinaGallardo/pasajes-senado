package services

import (
	"context"
	"errors"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"time"

	"fmt"

	"gorm.io/gorm"
)

type CupoService struct {
	repo          *repositories.CupoDerechoRepository
	userRepo      *repositories.UsuarioRepository
	itemRepo      *repositories.CupoDerechoItemRepository
	solicitudRepo *repositories.SolicitudRepository
}

type CupoInfo struct {
	Total        int
	Usado        int
	Saldo        int
	EsDisponible bool
	Mensaje      string
}

func NewCupoService(
	repo *repositories.CupoDerechoRepository,
	userRepo *repositories.UsuarioRepository,
	itemRepo *repositories.CupoDerechoItemRepository,
	solicitudRepo *repositories.SolicitudRepository,
) *CupoService {
	return &CupoService{
		repo:          repo,
		userRepo:      userRepo,
		itemRepo:      itemRepo,
		solicitudRepo: solicitudRepo,
	}
}

func (s *CupoService) CalcularCupo(ctx context.Context, usuarioID string, fecha time.Time) (*CupoInfo, error) {
	gestion := fecha.Year()
	mes := int(fecha.Month())

	user, err := s.userRepo.FindByID(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	if user.Tipo != "SENADOR_TITULAR" {
		return &CupoInfo{
			Total:        0,
			Usado:        0,
			Saldo:        0,
			EsDisponible: true,
			Mensaje:      "Usuario no es Senador Titular, no aplica cupos",
		}, nil
	}

	cupo, err := s.repo.FindByTitularAndPeriodo(ctx, usuarioID, gestion, mes)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := s.EnsureUserCuposDerecho(ctx, usuarioID, gestion, mes); err != nil {
				return nil, err
			}
			cupo, _ = s.repo.FindByTitularAndPeriodo(ctx, usuarioID, gestion, mes)
		} else {
			return nil, err
		}
	}

	if cupo == nil {
		return &CupoInfo{EsDisponible: false, Mensaje: "No se encontró configuración de cupos"}, nil
	}

	items, err := s.itemRepo.FindByCupoDerechoID(ctx, cupo.ID)
	if err != nil {
		return nil, err
	}

	cupo.Items = items
	saldo := cupo.GetSaldo()

	return &CupoInfo{
		Total:        cupo.CupoTotal,
		Usado:        cupo.CupoUsado,
		Saldo:        saldo,
		EsDisponible: saldo > 0,
		Mensaje:      "",
	}, nil
}

func (s *CupoService) EnsureUserCuposDerecho(ctx context.Context, usuarioID string, gestion int, mes int) error {
	user, err := s.userRepo.FindByID(ctx, usuarioID)
	if err != nil {
		return err
	}
	if user.Tipo == "SENADOR_TITULAR" {
		return s.generateCuposDerechoForSenador(ctx, user, gestion, mes)
	}
	return nil
}

func (s *CupoService) GenerateCuposDerechoForMonth(ctx context.Context, gestion int, mes int) error {
	users, err := s.userRepo.FindByRoleType(ctx, "SENADOR")
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.Tipo == "SENADOR_TITULAR" {
			if err := s.generateCuposDerechoForSenador(ctx, &user, gestion, mes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *CupoService) generateCuposDerechoForSenador(ctx context.Context, user *models.Usuario, gestion int, mes int) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.CupoDerechoRepository, tx *gorm.DB) error {
		return s.generateCuposDerechoTx(ctx, repoTx, s.itemRepo.WithTx(tx), user, gestion, mes)
	})
}

func (s *CupoService) generateCuposDerechoTx(ctx context.Context, cupoRepoTx *repositories.CupoDerechoRepository, itemRepoTx *repositories.CupoDerechoItemRepository, user *models.Usuario, gestion, mes int) error {
	weeksInfo := utils.GetWeeksInMonth(gestion, time.Month(mes))
	semanas := len(weeksInfo)
	if semanas == 0 {
		return fmt.Errorf("error calculando semanas para %d/%d", mes, gestion)
	}

	cupo, err := cupoRepoTx.WithContext(ctx).FindByTitularAndPeriodo(ctx, user.ID, gestion, mes)
	if err != nil {
		newCupo := models.CupoDerecho{
			SenTitularID: user.ID,
			Gestion:      gestion,
			Mes:          mes,
			TotalSemanas: semanas,
			CupoTotal:    semanas,
		}
		if err := cupoRepoTx.WithContext(ctx).Create(ctx, &newCupo); err != nil {
			return err
		}
		cupo = &newCupo
	}

	items, err := itemRepoTx.WithContext(ctx).FindByCupoDerechoID(ctx, cupo.ID)
	if err != nil {
		return err
	}

	existingWeeks := make(map[string]bool)
	for _, it := range items {
		existingWeeks[it.Semana] = true
	}

	var newItems []models.CupoDerechoItem
	for i, w := range weeksInfo {
		semanaKey := fmt.Sprintf("Semana %d", i+1)
		if !existingWeeks[semanaKey] {
			it := models.CupoDerechoItem{
				CupoDerechoID:           cupo.ID,
				SenTitularID:            user.ID,
				SenAsignadoID:           user.ID,
				Semana:                  semanaKey,
				Gestion:                 gestion,
				Mes:                     mes,
				FechaDesde:              &w.Inicio,
				FechaHasta:              &w.Fin,
				EstadoCupoDerechoCodigo: "DISPONIBLE",
			}
			newItems = append(newItems, it)
		}
	}

	if len(newItems) > 0 {
		if err := itemRepoTx.WithContext(ctx).CreateInBatches(ctx, newItems, 100); err != nil {
			return err
		}
	}

	return nil
}

func (s *CupoService) TransferirCupoDerecho(ctx context.Context, itemID string, targetUserID string, motivo string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.CupoDerechoRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)
		item, err := itemRepoTx.WithContext(ctx).FindByID(ctx, itemID)
		if err != nil {
			return err
		}
		if item.EstadoCupoDerechoCodigo != "DISPONIBLE" {
			return errors.New("el cupo no está disponible para transferir")
		}

		item.EstadoCupoDerechoCodigo = "TRANSFERIDO"
		item.SenAsignadoID = targetUserID
		item.EsTransferido = true
		item.MotivoTransfer = motivo
		nowTransfer := time.Now()
		item.FechaTransfer = &nowTransfer

		if err := itemRepoTx.WithContext(ctx).Update(ctx, item); err != nil {
			return err
		}

		return s.syncCupoUsadoTx(ctx, repoTx, itemRepoTx, item.SenTitularID, item.Gestion, item.Mes)
	})
}

func (s *CupoService) RevertirTransferencia(ctx context.Context, itemID string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.CupoDerechoRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)
		item, err := itemRepoTx.WithContext(ctx).FindByID(ctx, itemID)
		if err != nil {
			return err
		}
		if item.EstadoCupoDerechoCodigo != "TRANSFERIDO" {
			return errors.New("el cupo no está en estado transferido")
		}

		item.EstadoCupoDerechoCodigo = "DISPONIBLE"
		item.SenAsignadoID = item.SenTitularID
		item.EsTransferido = false
		item.MotivoTransfer = ""
		item.FechaTransfer = nil

		if err := itemRepoTx.WithContext(ctx).Update(ctx, item); err != nil {
			return err
		}

		return s.syncCupoUsadoTx(ctx, repoTx, itemRepoTx, item.SenTitularID, item.Gestion, item.Mes)
	})
}

func (s *CupoService) ProcesarConsumoPasaje(ctx context.Context, usuarioID string, gestion, mes int) error {
	return s.SyncCupoUsado(ctx, usuarioID, gestion, mes)
}

func (s *CupoService) GetAllByPeriodo(ctx context.Context, gestion, mes int) ([]models.CupoDerecho, error) {
	return s.repo.FindByPeriodo(ctx, gestion, mes)
}

func (s *CupoService) GetByID(ctx context.Context, id string) (*models.CupoDerecho, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *CupoService) GetAllCuposDerechoByPeriodo(ctx context.Context, gestion, mes int) ([]models.CupoDerechoItem, error) {
	return s.itemRepo.FindByPeriodo(ctx, gestion, mes)
}

func (s *CupoService) GetCupo(ctx context.Context, usuarioID string, gestion, mes int) (*models.CupoDerecho, error) {
	return s.repo.FindByTitularAndPeriodo(ctx, usuarioID, gestion, mes)
}

func (s *CupoService) IncrementarUso(ctx context.Context, usuarioID string, gestion, mes int) error {
	cupo, err := s.repo.FindByTitularAndPeriodo(ctx, usuarioID, gestion, mes)
	if err != nil {
		return err
	}
	cupo.CupoUsado++
	return s.repo.Update(ctx, cupo)
}

func (s *CupoService) RevertirUso(ctx context.Context, usuarioID string, gestion, mes int) error {
	cupo, err := s.repo.FindByTitularAndPeriodo(ctx, usuarioID, gestion, mes)
	if err != nil {
		return err
	}
	if cupo.CupoUsado > 0 {
		cupo.CupoUsado--
		return s.repo.Update(ctx, cupo)
	}
	return nil
}

func (s *CupoService) ResetCuposDerechoForMonth(ctx context.Context, gestion, mes int) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.CupoDerechoRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)
		items, err := itemRepoTx.WithContext(ctx).FindByPeriodo(ctx, gestion, mes)
		if err != nil {
			return err
		}
		for _, it := range items {
			if err := itemRepoTx.WithContext(ctx).DeleteUnscoped(ctx, &it); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *CupoService) SyncUsoForPeriod(ctx context.Context, gestion, mes int) error {
	cupos, err := s.repo.FindByPeriodo(ctx, gestion, mes)
	if err != nil {
		return err
	}

	for _, c := range cupos {
		if err := s.SyncCupoUsado(ctx, c.SenTitularID, gestion, mes); err != nil {
			return err
		}
	}
	return nil
}

func (s *CupoService) SyncCupoUsado(ctx context.Context, senadorID string, gestion, mes int) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.CupoDerechoRepository, tx *gorm.DB) error {
		return s.syncCupoUsadoTx(ctx, repoTx, s.itemRepo.WithTx(tx), senadorID, gestion, mes)
	})
}

func (s *CupoService) syncCupoUsadoTx(ctx context.Context, cupoRepo *repositories.CupoDerechoRepository, itemRepo *repositories.CupoDerechoItemRepository, senadorID string, gestion, mes int) error {
	cupo, err := cupoRepo.WithContext(ctx).FindByTitularAndPeriodo(ctx, senadorID, gestion, mes)
	if err != nil {
		return err
	}

	items, err := itemRepo.WithContext(ctx).FindByHolderAndPeriodo(ctx, senadorID, gestion, mes)
	if err != nil {
		return err
	}

	availableCount := 0
	for _, it := range items {
		if it.EstadoCupoDerechoCodigo == "DISPONIBLE" {
			availableCount++
		}
	}

	used := cupo.CupoTotal - availableCount
	if used < 0 {
		used = 0
	}

	if cupo.CupoUsado != used {
		cupo.CupoUsado = used
		return cupoRepo.WithContext(ctx).Update(ctx, cupo)
	}

	return nil
}

func (s *CupoService) GetCuposDerechoByCupoID(ctx context.Context, cupoID string) ([]models.CupoDerechoItem, error) {
	return s.itemRepo.FindByCupoDerechoID(ctx, cupoID)
}

func (s *CupoService) GetCuposDerechoByUsuario(ctx context.Context, authUserID string, gestion, mes int) ([]models.CupoDerechoItem, error) {
	user, err := s.userRepo.FindByID(ctx, authUserID)
	if err != nil {
		return nil, err
	}

	if user.Tipo != "SENADOR_TITULAR" {
		return s.itemRepo.FindForSuplenteByPeriodo(ctx, authUserID, gestion, mes)
	}

	return s.itemRepo.FindForTitularByPeriodo(ctx, authUserID, gestion, mes)
}

func (s *CupoService) GetCuposDerechoByUsuarioAndGestion(ctx context.Context, usuarioID string, gestion int) ([]models.CupoDerechoItem, error) {
	user, err := s.userRepo.FindByID(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	if user.Tipo != "SENADOR_TITULAR" {
		return s.itemRepo.FindForSuplenteByGestion(ctx, usuarioID, gestion)
	}

	return s.itemRepo.FindForTitularByGestion(ctx, usuarioID, gestion)
}

func (s *CupoService) GetCupoDerechoItemByID(ctx context.Context, id string) (*models.CupoDerechoItem, error) {
	return s.itemRepo.FindByID(ctx, id)
}

func (s *CupoService) GetCupoDerechoItemWeekDays(item *models.CupoDerechoItem) []map[string]string {
	return utils.GetWeekDays(item.FechaDesde, item.FechaHasta)
}

func (s *CupoService) GetSolicitudesByCupoItem(ctx context.Context, itemID string) ([]models.Solicitud, error) {
	return s.solicitudRepo.FindByCupoDerechoItemID(ctx, itemID)
}
