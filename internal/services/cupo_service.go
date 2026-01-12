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
	repo        *repositories.CupoRepository
	userRepo    *repositories.UsuarioRepository
	voucherRepo *repositories.AsignacionVoucherRepository
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
		repo:        repositories.NewCupoRepository(),
		userRepo:    repositories.NewUsuarioRepository(),
		voucherRepo: repositories.NewAsignacionVoucherRepository(),
	}
}

func (s *CupoService) CalcularCupo(ctx context.Context, usuarioID string, fecha time.Time) (*CupoInfo, error) {
	year := fecha.Year()
	month := int(fecha.Month())

	_ = s.EnsureUserVouchers(ctx, usuarioID, year, month)

	vouchers, err := s.voucherRepo.WithContext(ctx).FindByHolderAndPeriodo(usuarioID, year, month)

	if err != nil || len(vouchers) == 0 {
		return &CupoInfo{EsDisponible: false, Mensaje: "No tiene pasajes habilitados para este periodo."}, nil
	}

	var specificVoucher *models.AsignacionVoucher
	usados := 0
	total := len(vouchers)

	for i := range vouchers {
		v := &vouchers[i]
		if v.EstadoVoucherCodigo != "DISPONIBLE" {
			usados++
		}

		if v.FechaDesde != nil && v.FechaHasta != nil {
			if !fecha.Before(*v.FechaDesde) && !fecha.After(*v.FechaHasta) {
				specificVoucher = v
			}
		}
	}

	info := &CupoInfo{
		Total: total,
		Usado: usados,
		Saldo: total - usados,
	}

	if specificVoucher != nil {
		if specificVoucher.EstadoVoucherCodigo == "DISPONIBLE" {
			info.EsDisponible = true
			info.Mensaje = fmt.Sprintf("VÁLIDO: Corresponde a la %s (Vigente del %s al %s)",
				specificVoucher.Semana,
				specificVoucher.FechaDesde.Format("02/01"),
				specificVoucher.FechaHasta.Format("02/01"))
		} else {
			info.EsDisponible = false
			info.Mensaje = fmt.Sprintf("AGOTADO: El pasaje de la %s ya fue utilizado.", specificVoucher.Semana)
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

func (s *CupoService) EnsureUserVouchers(ctx context.Context, usuarioID string, gestion int, mes int) error {
	user, err := s.userRepo.WithContext(ctx).FindByID(usuarioID)
	if err != nil {
		return err
	}
	if user.Tipo == "SENADOR_TITULAR" {
		return s.generateVouchersForSenador(ctx, user, gestion, mes)
	}
	if user.Tipo == "SENADOR_SUPLENTE" && user.TitularID != nil {
		titular, err := s.userRepo.WithContext(ctx).FindByID(*user.TitularID)
		if err == nil {
			return s.generateVouchersForSenador(ctx, titular, gestion, mes)
		}
	}

	return nil
}

func (s *CupoService) GenerateVouchersForMonth(ctx context.Context, gestion int, mes int) error {
	senadores, err := s.userRepo.WithContext(ctx).FindAll()
	if err != nil {
		return err
	}

	for _, user := range senadores {
		if user.Tipo == "SENADOR_TITULAR" {
			_ = s.generateVouchersForSenador(ctx, &user, gestion, mes)
		}
	}

	return s.SyncUsoForPeriod(ctx, gestion, mes)
}

func (s *CupoService) generateVouchersForSenador(ctx context.Context, user *models.Usuario, gestion int, mes int) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.CupoRepository, tx *gorm.DB) error {
		return s.generateVouchersTx(repoTx, s.voucherRepo.WithTx(tx), user, gestion, mes)
	})
}

func (s *CupoService) generateVouchersTx(cupoRepoTx *repositories.CupoRepository, voucherRepoTx *repositories.AsignacionVoucherRepository, user *models.Usuario, gestion, mes int) error {
	weeksInfo := utils.GetWeeksInMonth(gestion, time.Month(mes))
	semanas := len(weeksInfo)
	if semanas == 0 {
		return fmt.Errorf("error calculando semanas para %d/%d", mes, gestion)
	}

	targetTotal := semanas

	cupo, err := cupoRepoTx.FindByTitularAndPeriodo(user.ID, gestion, mes)
	if err != nil {
		newCupo := models.Cupo{
			SenadorID:    user.ID,
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

	existingVouchers, _ := voucherRepoTx.FindByCupoID(cupo.ID)
	count := len(existingVouchers)

	if count < targetTotal {
		var newVouchers []models.AsignacionVoucher
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

			v := models.AsignacionVoucher{
				SenadorID:           user.ID,
				Gestion:             gestion,
				Mes:                 mes,
				Semana:              label,
				EstadoVoucherCodigo: "DISPONIBLE",
				CupoID:              cupo.ID,
				FechaDesde:          startDate,
				FechaHasta:          endDate,
			}
			newVouchers = append(newVouchers, v)
		}

		if len(newVouchers) > 0 {
			if err := voucherRepoTx.CreateInBatches(newVouchers, 100); err != nil {
				return err
			}
		}
	}

	return s.syncCupoUsadoTx(cupoRepoTx, voucherRepoTx, user.ID, gestion, mes)
}

func (s *CupoService) TransferirVoucher(ctx context.Context, voucherID string, destinoID string, motivo string) error {
	voucher, err := s.voucherRepo.WithContext(ctx).FindByID(voucherID)
	if err != nil {
		return errors.New("voucher no encontrado")
	}

	if voucher.EstadoVoucherCodigo != "DISPONIBLE" {
		return errors.New("el voucher no está disponible para transferencia (ya usado o transferido)")
	}

	voucher.EsTransferido = true
	voucher.BeneficiarioID = &destinoID
	voucher.FechaTransfer = utils.Ptr(time.Now())
	voucher.MotivoTransfer = motivo

	return s.voucherRepo.WithContext(ctx).Update(voucher)
}

func (s *CupoService) ProcesarConsumoPasaje(ctx context.Context, usuarioID string, gestion, mes int) error {
	return s.IncrementarUso(ctx, usuarioID, gestion, mes)
}

func (s *CupoService) GetAllByPeriodo(ctx context.Context, gestion, mes int) ([]models.Cupo, error) {
	return s.repo.WithContext(ctx).FindByPeriodo(gestion, mes)
}

func (s *CupoService) GetByID(ctx context.Context, id string) (*models.Cupo, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *CupoService) GetAllVouchersByPeriodo(ctx context.Context, gestion, mes int) ([]models.AsignacionVoucher, error) {
	repo := repositories.NewAsignacionVoucherRepository()
	return repo.WithContext(ctx).FindByPeriodo(gestion, mes)
}

func (s *CupoService) GetCupo(ctx context.Context, usuarioID string, gestion, mes int) (*models.Cupo, error) {
	return s.repo.WithContext(ctx).FindByTitularAndPeriodo(usuarioID, gestion, mes)
}

func (s *CupoService) IncrementarUso(ctx context.Context, usuarioID string, gestion, mes int) error {
	return s.repo.WithContext(ctx).RunTransaction(func(cupoRepoTx *repositories.CupoRepository, tx *gorm.DB) error {
		voucherRepoTx := s.voucherRepo.WithTx(tx)

		voucher, err := voucherRepoTx.FindAvailableByHolderAndPeriodo(usuarioID, gestion, mes)
		if err != nil {
			return errors.New("no hay pasajes disponibles para asignar (cupo agotado)")
		}

		voucher.EstadoVoucherCodigo = "USADO"
		if err := voucherRepoTx.Update(voucher); err != nil {
			return err
		}

		return s.syncCupoUsadoTx(cupoRepoTx, voucherRepoTx, voucher.SenadorID, gestion, mes)
	})
}

func (s *CupoService) RevertirUso(ctx context.Context, usuarioID string, gestion, mes int) error {
	return s.repo.WithContext(ctx).RunTransaction(func(cupoRepoTx *repositories.CupoRepository, tx *gorm.DB) error {
		voucherRepoTx := s.voucherRepo.WithTx(tx)

		vouchers, err := voucherRepoTx.FindByHolderAndPeriodo(usuarioID, gestion, mes)
		if err != nil {
			return err
		}

		for _, v := range vouchers {
			if v.EstadoVoucherCodigo == "USADO" {
				v.EstadoVoucherCodigo = "DISPONIBLE"
				if err := voucherRepoTx.Update(&v); err != nil {
					return err
				}
				return s.syncCupoUsadoTx(cupoRepoTx, voucherRepoTx, v.SenadorID, gestion, mes)
			}
		}
		return errors.New("no se encontró uso de pasaje para revertir")
	})
}

func (s *CupoService) ResetVouchersForMonth(ctx context.Context, gestion, mes int) error {
	return s.GenerateVouchersForMonth(ctx, gestion, mes)
}

func (s *CupoService) SyncUsoForPeriod(ctx context.Context, gestion, mes int) error {
	cupos, err := s.repo.WithContext(ctx).FindByPeriodo(gestion, mes)
	if err != nil {
		return err
	}

	for _, c := range cupos {
		if err := s.SyncCupoUsado(ctx, c.SenadorID, gestion, mes); err != nil {
			fmt.Printf("Error sincronizando uso para %s: %v\n", c.SenadorID, err)
		}
	}
	return nil
}

func (s *CupoService) SyncCupoUsado(ctx context.Context, senadorID string, gestion, mes int) error {
	return s.syncCupoUsadoTx(s.repo.WithContext(ctx), s.voucherRepo.WithContext(ctx), senadorID, gestion, mes)
}

func (s *CupoService) syncCupoUsadoTx(cupoRepo *repositories.CupoRepository, voucherRepo *repositories.AsignacionVoucherRepository, senadorID string, gestion, mes int) error {
	cupo, err := cupoRepo.FindByTitularAndPeriodo(senadorID, gestion, mes)
	if err != nil {
		return err
	}

	vouchers, err := voucherRepo.FindByCupoID(cupo.ID)
	if err != nil {
		return err
	}

	usados := 0
	for _, v := range vouchers {
		if v.EstadoVoucherCodigo == "USADO" {
			usados++
		}
	}

	if cupo.CupoUsado != usados {
		cupo.CupoUsado = usados
		return cupoRepo.Update(cupo)
	}

	return nil
}

func (s *CupoService) GetVouchersByCupoID(ctx context.Context, cupoID string) ([]models.AsignacionVoucher, error) {
	return s.voucherRepo.WithContext(ctx).FindByCupoID(cupoID)
}

func (s *CupoService) GetVouchersByUsuario(ctx context.Context, usuarioID string, gestion, mes int) ([]models.AsignacionVoucher, error) {
	_ = s.EnsureUserVouchers(ctx, usuarioID, gestion, mes)
	return s.voucherRepo.WithContext(ctx).FindByHolderAndPeriodo(usuarioID, gestion, mes)
}

func (s *CupoService) GetVouchersByUsuarioAndGestion(ctx context.Context, usuarioID string, gestion int) ([]models.AsignacionVoucher, error) {
	return s.voucherRepo.WithContext(ctx).FindByHolderAndGestion(usuarioID, gestion)
}

func (s *CupoService) GetVoucherByID(ctx context.Context, id string) (*models.AsignacionVoucher, error) {
	return s.voucherRepo.WithContext(ctx).FindByID(id)
}

func (s *CupoService) GetVoucherWeekDays(voucher *models.AsignacionVoucher) []map[string]string {
	return utils.GetWeekDays(voucher.FechaDesde, voucher.FechaHasta)
}
