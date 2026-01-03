package services

import (
	"errors"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"time"

	"fmt"
)

type CupoService struct {
	repo     *repositories.CupoRepository
	userRepo *repositories.UsuarioRepository
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
		repo:     repositories.NewCupoRepository(),
		userRepo: repositories.NewUsuarioRepository(),
	}
}

func (s *CupoService) CalcularCupo(usuarioID string, fecha time.Time) (*CupoInfo, error) {
	year := fecha.Year()
	month := int(fecha.Month())

	_ = s.GenerateVouchersForMonth(year, month)

	voucherRepo := repositories.NewAsignacionVoucherRepository()
	vouchers, err := voucherRepo.FindByHolderAndPeriodo(usuarioID, year, month)

	if err != nil || len(vouchers) == 0 {
		return &CupoInfo{EsDisponible: false, Mensaje: "No tiene pasajes habilitados para este mes."}, nil
	}

	total := len(vouchers)
	usados := 0
	for _, v := range vouchers {
		if v.Estado != "DISPONIBLE" {
			usados++
		}
	}

	info := &CupoInfo{
		Total: total,
		Usado: usados,
		Saldo: total - usados,
	}

	if info.Saldo <= 0 {
		info.EsDisponible = false
		info.Mensaje = fmt.Sprintf("Cupo mensual agotado (%d/%d asignados)", usados, total)
	} else {
		info.EsDisponible = true
		info.Mensaje = fmt.Sprintf("Disponible (%d restantes)", info.Saldo)
	}

	return info, nil
}

type WeekRange struct {
	Inicio time.Time
	Fin    time.Time
}

func (s *CupoService) GetWeeksInMonth(year int, month time.Month) []WeekRange {
	t := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	lastDay := t.AddDate(0, 1, -1)

	var weeks []WeekRange
	for d := t; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Monday {
			monday := d
			sunday := d.AddDate(0, 0, 6)
			weeks = append(weeks, WeekRange{Inicio: monday, Fin: sunday})
		}
	}
	return weeks
}

func (s *CupoService) GenerateVouchersForMonth(gestion int, mes int) error {
	senadores, err := s.userRepo.FindAll()
	if err != nil {
		return err
	}
	weeksInfo := s.GetWeeksInMonth(gestion, time.Month(mes))
	semanas := len(weeksInfo)
	if semanas == 0 {
		return errors.New("error calculando semanas")
	}

	voucherRepo := repositories.NewAsignacionVoucherRepository()

	for _, user := range senadores {
		if user.Tipo != "SENADOR_TITULAR" {
			continue
		}
		targetTotal := semanas

		// 1. Asegurar que el Cupo exista y esté actualizado
		cupo, err := s.repo.FindByTitularAndPeriodo(user.ID, gestion, mes)
		if err != nil {
			newCupo := models.Cupo{
				SenadorID:    user.ID,
				Gestion:      gestion,
				Mes:          mes,
				TotalSemanas: semanas,
				CupoTotal:    targetTotal,
			}
			if err := s.repo.Create(&newCupo); err != nil {
				return fmt.Errorf("error creando cupo mensual para %s: %w", user.Username, err)
			}
			cupo = &newCupo
		} else {
			if cupo.TotalSemanas != semanas || cupo.CupoTotal != targetTotal {
				cupo.TotalSemanas = semanas
				cupo.CupoTotal = targetTotal
				if err := s.repo.Update(cupo); err != nil {
					return fmt.Errorf("error actualizando cupo para %s: %w", user.Username, err)
				}
			}
		}

		// 2. Sincronizar Vouchers
		existingVouchers, _ := voucherRepo.FindByCupoID(cupo.ID)
		count := len(existingVouchers)

		if count < targetTotal {
			for i := count; i < targetTotal; i++ {
				weekNum := i + 1
				label := fmt.Sprintf("SEMANA %d", weekNum)

				if weekNum == semanas {
					label = fmt.Sprintf("SEMANA %d (REGIONAL)", weekNum)
				}

				var startDate, endDate *time.Time
				if i < len(weeksInfo) {
					sDate := weeksInfo[i].Inicio
					eDate := weeksInfo[i].Fin
					startDate = &sDate
					endDate = &eDate
				}

				v := models.AsignacionVoucher{
					SenadorID:  user.ID,
					Gestion:    gestion,
					Mes:        mes,
					Semana:     label,
					Estado:     "DISPONIBLE",
					CupoID:     cupo.ID,
					FechaDesde: startDate,
					FechaHasta: endDate,
				}
				if err := voucherRepo.Create(&v); err != nil {
					return fmt.Errorf("error creando voucher adicional para %s: %w", user.Username, err)
				}
			}
		}
	}

	// 3. Sincronizar saldos de uso para todos
	return s.SyncUsoForPeriod(gestion, mes)
}

func (s *CupoService) TransferirVoucher(voucherID string, destinoID string, motivo string) error {
	repo := repositories.NewAsignacionVoucherRepository()

	voucher, err := repo.FindByID(voucherID)
	if err != nil {
		return errors.New("voucher no encontrado")
	}

	if voucher.Estado != "DISPONIBLE" {
		return errors.New("el voucher no está disponible para transferencia (ya usado o transferido)")
	}

	now := time.Now()

	voucher.EsTransferido = true
	voucher.BeneficiarioID = &destinoID
	voucher.FechaTransfer = &now
	voucher.MotivoTransfer = motivo

	return repo.Update(voucher)
}

func (s *CupoService) ProcesarConsumoPasaje(usuarioID string, gestion, mes int) error {
	return s.IncrementarUso(usuarioID, gestion, mes)
}

func (s *CupoService) GetAllByPeriodo(gestion, mes int) ([]models.Cupo, error) {
	return s.repo.FindByPeriodo(gestion, mes)
}

func (s *CupoService) GetAllVouchersByPeriodo(gestion, mes int) ([]models.AsignacionVoucher, error) {
	repo := repositories.NewAsignacionVoucherRepository()
	return repo.FindByPeriodo(gestion, mes)
}

func (s *CupoService) GetCupo(usuarioID string, gestion, mes int) (*models.Cupo, error) {
	return s.repo.FindByTitularAndPeriodo(usuarioID, gestion, mes)
}

func (s *CupoService) IncrementarUso(usuarioID string, gestion, mes int) error {
	voucherRepo := repositories.NewAsignacionVoucherRepository()

	voucher, err := voucherRepo.FindAvailableByHolderAndPeriodo(usuarioID, gestion, mes)
	if err != nil {
		_ = s.GenerateVouchersForMonth(gestion, mes)
		voucher, err = voucherRepo.FindAvailableByHolderAndPeriodo(usuarioID, gestion, mes)
		if err != nil {
			return errors.New("no hay pasajes disponibles para asignar (cupo agotado)")
		}
	}

	voucher.Estado = "USADO"
	if err := voucherRepo.Update(voucher); err != nil {
		return err
	}

	// Sincronizar el cupo
	return s.SyncCupoUsado(voucher.SenadorID, gestion, mes)
}

func (s *CupoService) RevertirUso(usuarioID string, gestion, mes int) error {
	voucherRepo := repositories.NewAsignacionVoucherRepository()
	vouchers, err := voucherRepo.FindByHolderAndPeriodo(usuarioID, gestion, mes)
	if err != nil {
		return err
	}

	for _, v := range vouchers {
		if v.Estado == "USADO" {
			v.Estado = "DISPONIBLE"
			if err := voucherRepo.Update(&v); err != nil {
				return err
			}
			return s.SyncCupoUsado(v.SenadorID, gestion, mes)
		}
	}
	return errors.New("no se encontró uso de pasaje para revertir")
}

func (s *CupoService) ResetVouchersForMonth(gestion, mes int) error {
	// Ahora "Reset" significa sincronizar y asegurar que todo esté correcto según las reglas
	return s.GenerateVouchersForMonth(gestion, mes)
}

func (s *CupoService) SyncUsoForPeriod(gestion, mes int) error {
	cupos, err := s.repo.FindByPeriodo(gestion, mes)
	if err != nil {
		return err
	}

	for _, c := range cupos {
		if err := s.SyncCupoUsado(c.SenadorID, gestion, mes); err != nil {
			fmt.Printf("Error sincronizando uso para %s: %v\n", c.SenadorID, err)
		}
	}
	return nil
}

func (s *CupoService) SyncCupoUsado(senadorID string, gestion, mes int) error {
	cupo, err := s.repo.FindByTitularAndPeriodo(senadorID, gestion, mes)
	if err != nil {
		return err
	}

	voucherRepo := repositories.NewAsignacionVoucherRepository()
	vouchers, err := voucherRepo.FindByCupoID(cupo.ID)
	if err != nil {
		return err
	}

	usados := 0
	for _, v := range vouchers {
		if v.Estado == "USADO" {
			usados++
		}
	}

	if cupo.CupoUsado != usados {
		cupo.CupoUsado = usados
		return s.repo.Update(cupo)
	}

	return nil
}

func (s *CupoService) GetVouchersByCupoID(cupoID string) ([]models.AsignacionVoucher, error) {
	repo := repositories.NewAsignacionVoucherRepository()
	return repo.FindByCupoID(cupoID)
}
