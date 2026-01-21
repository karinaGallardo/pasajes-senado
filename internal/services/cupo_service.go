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
	repo     *repositories.CupoDerechoRepository
	userRepo *repositories.UsuarioRepository
	itemRepo *repositories.CupoDerechoItemRepository
}

type CupoInfo struct {
	Total        int
	Usado        int
	Saldo        int
	EsDisponible bool
	Mensaje      string
}

func NewCupoService() *CupoService {
	return &CupoService{
		repo:     repositories.NewCupoDerechoRepository(),
		userRepo: repositories.NewUsuarioRepository(),
		itemRepo: repositories.NewCupoDerechoItemRepository(),
	}
}

func (s *CupoService) CalcularCupo(ctx context.Context, usuarioID string, fecha time.Time) (*CupoInfo, error) {
	year := fecha.Year()
	month := int(fecha.Month())

	_ = s.EnsureUserCuposDerecho(ctx, usuarioID, year, month)

	items, err := s.itemRepo.WithContext(ctx).FindByHolderAndPeriodo(usuarioID, year, month)

	if err != nil || len(items) == 0 {
		return &CupoInfo{EsDisponible: false, Mensaje: "No tiene pasajes habilitados para este periodo."}, nil
	}

	var specificItem *models.CupoDerechoItem
	usados := 0
	total := len(items)

	for i := range items {
		v := &items[i]
		if v.EstadoCupoDerechoCodigo != "DISPONIBLE" {
			usados++
		}

		if v.FechaDesde != nil && v.FechaHasta != nil {
			if !fecha.Before(*v.FechaDesde) && !fecha.After(*v.FechaHasta) {
				specificItem = v
			}
		}
	}

	info := &CupoInfo{
		Total: total,
		Usado: usados,
		Saldo: total - usados,
	}

	if specificItem != nil {
		if specificItem.EstadoCupoDerechoCodigo == "DISPONIBLE" {
			info.EsDisponible = true
			info.Mensaje = fmt.Sprintf("VÁLIDO: Corresponde a la %s (Vigente del %s al %s)",
				specificItem.Semana,
				specificItem.FechaDesde.Format("02/01"),
				specificItem.FechaHasta.Format("02/01"))
		} else {
			info.EsDisponible = false
			info.Mensaje = fmt.Sprintf("AGOTADO: El pasaje de la %s ya fue utilizado.", specificItem.Semana)
		}
	} else {
		if info.Saldo > 0 {
			info.EsDisponible = true
			info.Mensaje = fmt.Sprintf("Disponible (%d restantes en el mes). Nota: La fecha no coincide con el rango exacto de una semana, pero se puede asignar.", info.Saldo)
		} else {
			info.EsDisponible = false
			info.Mensaje = fmt.Sprintf("Cupo agotado (%d/%d usados en el mes).", usados, total)
		}
	}

	return info, nil
}

func (s *CupoService) EnsureUserCuposDerecho(ctx context.Context, usuarioID string, gestion int, mes int) error {
	user, err := s.userRepo.WithContext(ctx).FindByID(usuarioID)
	if err != nil {
		return err
	}

	if user.Tipo == "SENADOR_TITULAR" {
		return s.generateCuposDerechoForSenador(ctx, user, gestion, mes)
	}

	if user.Tipo == "SENADOR_SUPLENTE" && user.TitularID != nil {
		titular, err := s.userRepo.WithContext(ctx).FindByID(*user.TitularID)
		if err == nil && titular != nil {
			return s.generateCuposDerechoForSenador(ctx, titular, gestion, mes)
		}
	}

	return nil
}

func (s *CupoService) GenerateCuposDerechoForMonth(ctx context.Context, gestion int, mes int) error {
	users, err := s.userRepo.WithContext(ctx).FindAllSenators()
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.Tipo == "SENADOR_TITULAR" {
			_ = s.generateCuposDerechoForSenador(ctx, &user, gestion, mes)
		}
	}

	return s.SyncUsoForPeriod(ctx, gestion, mes)
}

func (s *CupoService) generateCuposDerechoForSenador(ctx context.Context, user *models.Usuario, gestion int, mes int) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.CupoDerechoRepository, tx *gorm.DB) error {
		return s.generateCuposDerechoTx(repoTx, s.itemRepo.WithTx(tx), user, gestion, mes)
	})
}

func (s *CupoService) generateCuposDerechoTx(cupoRepoTx *repositories.CupoDerechoRepository, itemRepoTx *repositories.CupoDerechoItemRepository, user *models.Usuario, gestion, mes int) error {
	weeksInfo := utils.GetWeeksInMonth(gestion, time.Month(mes))
	semanas := len(weeksInfo)
	if semanas == 0 {
		return fmt.Errorf("error calculando semanas para %d/%d", mes, gestion)
	}

	targetTotal := semanas

	cupo, err := cupoRepoTx.FindByTitularAndPeriodo(user.ID, gestion, mes)
	if err != nil {
		newCupo := models.CupoDerecho{
			SenTitularID: user.ID,
			Gestion:      gestion,
			Mes:          mes,
			TotalSemanas: semanas,
			CupoTotal:    targetTotal,
		}
		if err := cupoRepoTx.Create(&newCupo); err != nil {
			return err
		}
		cupo = &newCupo
	} else {
		if cupo.TotalSemanas != semanas || cupo.CupoTotal != targetTotal {
			cupo.TotalSemanas = semanas
			cupo.CupoTotal = targetTotal
			cupoRepoTx.Update(cupo)
		}
	}

	existingItems, _ := itemRepoTx.FindByCupoDerechoID(cupo.ID)
	count := len(existingItems)

	if count < targetTotal {
		var newItems []models.CupoDerechoItem
		for i := count; i < targetTotal; i++ {
			weekNum := i + 1
			label := fmt.Sprintf("SEMANA %d", weekNum)
			if weekNum == semanas {
				label = fmt.Sprintf("SEMANA %d (REGIONAL)", weekNum)
			}

			var startDate, endDate *time.Time
			if i < len(weeksInfo) {
				startDate = &weeksInfo[i].Inicio
				endDate = &weeksInfo[i].Fin
			}

			v := models.CupoDerechoItem{
				SenTitularID:            user.ID,
				SenAsignadoID:           user.ID,
				Gestion:                 gestion,
				Mes:                     mes,
				Semana:                  label,
				EstadoCupoDerechoCodigo: "DISPONIBLE",
				CupoDerechoID:           cupo.ID,
				FechaDesde:              startDate,
				FechaHasta:              endDate,
			}
			newItems = append(newItems, v)
		}

		if len(newItems) > 0 {
			if err := itemRepoTx.CreateInBatches(newItems, 100); err != nil {
				return err
			}
		}
	}

	return s.syncCupoUsadoTx(cupoRepoTx, itemRepoTx, user.ID, gestion, mes)
}

func (s *CupoService) TransferirCupoDerecho(ctx context.Context, itemID string, destinoID string, motivo string) error {
	item, err := s.itemRepo.WithContext(ctx).FindByID(itemID)
	if err != nil {
		return errors.New("derecho no encontrado")
	}

	if item.EstadoCupoDerechoCodigo != "DISPONIBLE" {
		return errors.New("el derecho no está disponible para transferencia (ya usado o transferido)")
	}

	item.EsTransferido = true
	item.SenAsignadoID = destinoID
	item.FechaTransfer = utils.Ptr(time.Now())
	item.MotivoTransfer = motivo

	return s.itemRepo.WithContext(ctx).Update(item)
}

func (s *CupoService) RevertirTransferencia(ctx context.Context, itemID string) error {
	item, err := s.itemRepo.WithContext(ctx).FindByID(itemID)
	if err != nil {
		return errors.New("derecho no encontrado")
	}

	if !item.EsTransferido {
		return errors.New("el derecho no ha sido transferido")
	}

	if item.EstadoCupoDerechoCodigo != "DISPONIBLE" {
		return errors.New("no se puede revertir: el derecho ya fue utilizado por el beneficiario")
	}

	item.EsTransferido = false
	item.SenAsignadoID = item.SenTitularID
	item.FechaTransfer = nil
	item.MotivoTransfer = ""

	return s.itemRepo.WithContext(ctx).Update(item)
}

func (s *CupoService) ProcesarConsumoPasaje(ctx context.Context, usuarioID string, gestion, mes int) error {
	return s.IncrementarUso(ctx, usuarioID, gestion, mes)
}

func (s *CupoService) GetAllByPeriodo(ctx context.Context, gestion, mes int) ([]models.CupoDerecho, error) {
	return s.repo.WithContext(ctx).FindByPeriodo(gestion, mes)
}

func (s *CupoService) GetByID(ctx context.Context, id string) (*models.CupoDerecho, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *CupoService) GetAllCuposDerechoByPeriodo(ctx context.Context, gestion, mes int) ([]models.CupoDerechoItem, error) {
	return s.itemRepo.WithContext(ctx).FindByPeriodo(gestion, mes)
}

func (s *CupoService) GetCupo(ctx context.Context, usuarioID string, gestion, mes int) (*models.CupoDerecho, error) {
	return s.repo.WithContext(ctx).FindByTitularAndPeriodo(usuarioID, gestion, mes)
}

func (s *CupoService) IncrementarUso(ctx context.Context, usuarioID string, gestion, mes int) error {
	return s.repo.WithContext(ctx).RunTransaction(func(cupoRepoTx *repositories.CupoDerechoRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		item, err := itemRepoTx.FindAvailableByHolderAndPeriodo(usuarioID, gestion, mes)
		if err != nil {
			return errors.New("no hay pasajes disponibles para asignar (cupo agotado)")
		}

		item.EstadoCupoDerechoCodigo = "USADO"
		if err := itemRepoTx.Update(item); err != nil {
			return err
		}

		return s.syncCupoUsadoTx(cupoRepoTx, itemRepoTx, item.SenTitularID, gestion, mes)
	})
}

func (s *CupoService) RevertirUso(ctx context.Context, usuarioID string, gestion, mes int) error {
	return s.repo.WithContext(ctx).RunTransaction(func(cupoRepoTx *repositories.CupoDerechoRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		items, err := itemRepoTx.FindByHolderAndPeriodo(usuarioID, gestion, mes)
		if err != nil {
			return err
		}

		for _, v := range items {
			if v.EstadoCupoDerechoCodigo == "USADO" {
				v.EstadoCupoDerechoCodigo = "DISPONIBLE"
				if err := itemRepoTx.Update(&v); err != nil {
					return err
				}
				return s.syncCupoUsadoTx(cupoRepoTx, itemRepoTx, v.SenTitularID, gestion, mes)
			}
		}
		return errors.New("no se encontró uso de pasaje para revertir")
	})
}

func (s *CupoService) ResetCuposDerechoForMonth(ctx context.Context, gestion, mes int) error {
	return s.GenerateCuposDerechoForMonth(ctx, gestion, mes)
}

func (s *CupoService) SyncUsoForPeriod(ctx context.Context, gestion, mes int) error {
	cupos, err := s.repo.WithContext(ctx).FindByPeriodo(gestion, mes)
	if err != nil {
		return err
	}

	for _, c := range cupos {
		if err := s.SyncCupoUsado(ctx, c.SenTitularID, gestion, mes); err != nil {
			fmt.Printf("Error sincronizando uso para %s: %v\n", c.SenTitularID, err)
		}
	}
	return nil
}

func (s *CupoService) SyncCupoUsado(ctx context.Context, senadorID string, gestion, mes int) error {
	return s.syncCupoUsadoTx(s.repo.WithContext(ctx), s.itemRepo.WithContext(ctx), senadorID, gestion, mes)
}

func (s *CupoService) syncCupoUsadoTx(cupoRepo *repositories.CupoDerechoRepository, itemRepo *repositories.CupoDerechoItemRepository, senadorID string, gestion, mes int) error {
	cupo, err := cupoRepo.FindByTitularAndPeriodo(senadorID, gestion, mes)
	if err != nil {
		return err
	}

	items, err := itemRepo.FindByCupoDerechoID(cupo.ID)
	if err != nil {
		return err
	}

	usados := 0
	for _, v := range items {
		if v.EstadoCupoDerechoCodigo == "USADO" {
			usados++
		}
	}

	if cupo.CupoUsado != usados {
		cupo.CupoUsado = usados
		return cupoRepo.Update(cupo)
	}

	return nil
}

func (s *CupoService) GetCuposDerechoByCupoID(ctx context.Context, cupoID string) ([]models.CupoDerechoItem, error) {
	return s.itemRepo.WithContext(ctx).FindByCupoDerechoID(cupoID)
}

func (s *CupoService) GetCuposDerechoByUsuario(ctx context.Context, usuarioID string, gestion, mes int) ([]models.CupoDerechoItem, error) {
	user, err := s.userRepo.WithContext(ctx).FindByID(usuarioID)
	if err != nil {
		return nil, err
	}

	if user.Tipo == "SENADOR_TITULAR" {
		_ = s.EnsureUserCuposDerecho(ctx, usuarioID, gestion, mes)
		return s.itemRepo.WithContext(ctx).FindForTitularByPeriodo(usuarioID, gestion, mes)
	}

	if user.Tipo == "SENADOR_SUPLENTE" && user.TitularID != nil {
		titular, err := s.userRepo.WithContext(ctx).FindByID(*user.TitularID)
		if err == nil && titular != nil {
			_ = s.EnsureUserCuposDerecho(ctx, titular.ID, gestion, mes)
		}
		return s.itemRepo.WithContext(ctx).FindForSuplenteByPeriodo(usuarioID, gestion, mes)
	}

	_ = s.EnsureUserCuposDerecho(ctx, usuarioID, gestion, mes)
	return s.itemRepo.WithContext(ctx).FindByHolderAndPeriodo(usuarioID, gestion, mes)
}

func (s *CupoService) GetCuposDerechoByUsuarioAndGestion(ctx context.Context, usuarioID string, gestion int) ([]models.CupoDerechoItem, error) {
	user, err := s.userRepo.WithContext(ctx).FindByID(usuarioID)
	if err != nil {
		return nil, err
	}

	if user.Tipo == "SENADOR_SUPLENTE" {
		return s.itemRepo.WithContext(ctx).FindForSuplenteByGestion(usuarioID, gestion)
	}

	return s.itemRepo.WithContext(ctx).FindForTitularByGestion(usuarioID, gestion)
}

func (s *CupoService) GetCupoDerechoItemByID(ctx context.Context, id string) (*models.CupoDerechoItem, error) {
	return s.itemRepo.WithContext(ctx).FindByID(id)
}

func (s *CupoService) GetCupoDerechoItemWeekDays(item *models.CupoDerechoItem) []map[string]string {
	return utils.GetWeekDays(item.FechaDesde, item.FechaHasta)
}
