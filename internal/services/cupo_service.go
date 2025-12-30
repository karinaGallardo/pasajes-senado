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
	vouchers, err := voucherRepo.FindByUsuarioAndPeriodo(usuarioID, year, month)

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

func (s *CupoService) CountMondaysInMonth(year int, month time.Month) int {
	t := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	lastDay := t.AddDate(0, 1, -1)

	count := 0
	for d := t; !d.After(lastDay); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Monday {
			count++
		}
	}
	return count
}

func (s *CupoService) GenerateVouchersForMonth(gestion int, mes int) error {
	senadores, err := s.userRepo.FindAll()
	if err != nil {
		return err
	}
	semanas := s.CountMondaysInMonth(gestion, time.Month(mes))
	if semanas == 0 {
		return errors.New("error calculando semanas")
	}

	voucherRepo := repositories.NewAsignacionVoucherRepository()

	for _, user := range senadores {
		existing, _ := voucherRepo.FindByUsuarioAndPeriodo(user.ID, gestion, mes)
		if len(existing) > 0 {
			continue
		}

		if user.Tipo == "SENADOR_TITULAR" {
			count := semanas - 1
			for i := 1; i <= count; i++ {
				v := models.AsignacionVoucher{
					UsuarioID: user.ID,
					Gestion:   gestion,
					Mes:       mes,
					Semana:    fmt.Sprintf("SEMANA %d", i),
					Estado:    "DISPONIBLE",
				}
				voucherRepo.Create(&v)
			}
		} else if user.Tipo == "SENADOR_SUPLENTE" {
			v := models.AsignacionVoucher{
				UsuarioID: user.ID,
				Gestion:   gestion,
				Mes:       mes,
				Semana:    fmt.Sprintf("SEMANA %d (REGIONAL)", semanas),
				Estado:    "DISPONIBLE",
			}
			voucherRepo.Create(&v)
		}
	}
	return nil
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

	origenID := voucher.UsuarioID
	now := time.Now()

	voucher.EsTransferido = true
	voucher.UsuarioOrigenID = &origenID
	voucher.UsuarioID = destinoID
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
	return s.repo.FindByUsuarioAndPeriodo(usuarioID, gestion, mes)
}

func (s *CupoService) IncrementarUso(usuarioID string, gestion, mes int) error {
	voucherRepo := repositories.NewAsignacionVoucherRepository()

	voucher, err := voucherRepo.FindAvailableByUsuarioAndPeriodo(usuarioID, gestion, mes)
	if err != nil {
		_ = s.GenerateVouchersForMonth(gestion, mes)
		voucher, err = voucherRepo.FindAvailableByUsuarioAndPeriodo(usuarioID, gestion, mes)
		if err != nil {
			return errors.New("no hay pasajes disponibles para asignar (cupo agotado)")
		}
	}

	voucher.Estado = "USADO"
	return voucherRepo.Update(voucher)
}

func (s *CupoService) RevertirUso(usuarioID string, gestion, mes int) error {
	voucherRepo := repositories.NewAsignacionVoucherRepository()
	vouchers, err := voucherRepo.FindByUsuarioAndPeriodo(usuarioID, gestion, mes)
	if err != nil {
		return err
	}

	for _, v := range vouchers {
		if v.Estado == "USADO" {
			v.Estado = "DISPONIBLE"
			return voucherRepo.Update(&v)
		}
	}
	return errors.New("no se encontró uso de pasaje para revertir")
}

func (s *CupoService) ResetVouchersForMonth(gestion, mes int) error {
	repo := repositories.NewAsignacionVoucherRepository()
	vouchers, _ := repo.FindByPeriodo(gestion, mes)
	for _, v := range vouchers {
		if v.Estado == "USADO" {
			return fmt.Errorf("no se puede reiniciar el mes: hay pasajes ya usuados (Usuario: %s)", v.UsuarioID)
		}
	}
	for _, v := range vouchers {
		repo.DeleteUnscoped(&v)
	}
	return nil
}
