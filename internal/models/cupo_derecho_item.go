package models

import (
	"fmt"
	"time"
)

type CupoDerechoItem struct {
	BaseModel
	SenTitularID string   `gorm:"size:36;not null;index;comment:Senador dueño del cupo por derecho"`
	SenTitular   *Usuario `gorm:"foreignKey:SenTitularID"`

	SenAsignadoID string   `gorm:"size:36;not null;index;comment:Senador que tiene el derecho de uso actual"`
	SenAsignado   *Usuario `gorm:"foreignKey:SenAsignadoID"`

	EsTransferido  bool       `gorm:"default:false;comment:Indica si el derecho ha sido transferido"`
	FechaTransfer  *time.Time `gorm:"type:timestamp;comment:Fecha de la transferencia"`
	MotivoTransfer string     `gorm:"size:255;comment:Motivo de la transferencia"`

	Gestion int    `gorm:"not null;index"`
	Mes     int    `gorm:"not null;index"`
	Semana  string `gorm:"size:50;index"`

	CupoDerechoID string       `gorm:"size:36;index;comment:ID del cupo por derecho"`
	CupoDerecho   *CupoDerecho `gorm:"foreignKey:CupoDerechoID"`

	EstadoCupoDerechoCodigo string             `gorm:"size:50;default:'DISPONIBLE';index" json:"Estado"`
	EstadoCupoDerecho       *EstadoCupoDerecho `gorm:"foreignKey:EstadoCupoDerechoCodigo"`

	Solicitudes []Solicitud `gorm:"foreignKey:CupoDerechoItemID"`

	FechaDesde *time.Time `gorm:"type:timestamp;comment:Fecha desde la cual el cupo es válido"`
	FechaHasta *time.Time `gorm:"type:timestamp;comment:Fecha hasta la cual el cupo es válido"`

	// Seq is an auto-incrementing field managed by DB to ensure atomic sequential ordering
	Seq int64 `gorm:"autoIncrement;not null;<-:false"`
}

func (CupoDerechoItem) TableName() string {
	return "cupo_derecho_items"
}

func (v CupoDerechoItem) getSolicitudIda() *Solicitud {
	for i := range v.Solicitudes {
		s := &v.Solicitudes[i]
		if s.EstadoSolicitudCodigo != nil && *s.EstadoSolicitudCodigo == "RECHAZADO" {
			continue
		}
		for _, t := range s.Items {
			if t.Tipo == TipoSolicitudItemIda {
				return s
			}
		}
	}
	return nil
}

func (v CupoDerechoItem) getSolicitudVuelta() *Solicitud {
	for i := range v.Solicitudes {
		s := &v.Solicitudes[i]
		if s.EstadoSolicitudCodigo != nil && *s.EstadoSolicitudCodigo == "RECHAZADO" {
			continue
		}
		for _, t := range s.Items {
			if t.Tipo == TipoSolicitudItemVuelta {
				return s
			}
		}
	}
	return nil
}
func (v CupoDerechoItem) GetSolicitud() *Solicitud {
	if s := v.getSolicitudIda(); s != nil {
		return s
	}
	return v.getSolicitudVuelta()
}

func (v CupoDerechoItem) GetDescargo() *Descargo {
	for i := range v.Solicitudes {
		if v.Solicitudes[i].Descargo != nil {
			return v.Solicitudes[i].Descargo
		}
	}
	return nil
}

func (v CupoDerechoItem) IsVencido() bool {
	now := time.Now()

	if now.Year() > v.Gestion {
		return true
	}

	if now.Year() == v.Gestion && int(now.Month()) > v.Mes {
		return true
	}

	return false
}

func (v CupoDerechoItem) IsActiveWeek() bool {
	if v.FechaDesde == nil || v.FechaHasta == nil {
		return false
	}
	now := time.Now()
	return now.After(*v.FechaDesde) && now.Before(v.FechaHasta.Add(24*time.Hour))
}

func (v CupoDerechoItem) IsDisponible() bool {
	return v.EstadoCupoDerechoCodigo == "DISPONIBLE"
}

func (v CupoDerechoItem) CanBeReverted() bool {
	for i := range v.Solicitudes {
		s := &v.Solicitudes[i]
		for j := range s.Items {
			it := &s.Items[j]
			estado := it.GetEstado()
			if estado == "APROBADO" || estado == "EMITIDO" {
				return false
			}
		}
	}
	return true
}

type CupoDerechoItemPermissions struct {
	CanTransfer        bool
	CanRevert          bool
	CanPrint           bool
	CanTomarCupo       bool
	CanAsignarCupo     bool
	CanCreate          bool
	CanCreateIdaVuelta bool
	CanEdit            bool
	CanView            bool
}

func (v CupoDerechoItem) GetPermissions(authUser *Usuario, targetUser *Usuario, targetHasQuota bool) CupoDerechoItemPermissions {
	perms := CupoDerechoItemPermissions{}

	isViewerAdminOrResponsable := authUser.IsAdminOrResponsable()
	isViewerSuplente := authUser != nil && (authUser.Tipo == "SENADOR_SUPLENTE")
	isTargetSuplente := targetUser != nil && (targetUser.Tipo == "SENADOR_SUPLENTE")
	isEncargado := targetUser != nil && targetUser.EncargadoID != nil && *targetUser.EncargadoID == authUser.ID

	isDisponible := v.IsDisponible()
	isVencido := v.IsVencido()
	isTransferido := v.EsTransferido
	isOwner := v.SenAsignadoID == targetUser.ID

	// 1. Admin Actions
	if isViewerAdminOrResponsable {
		if isTransferido && v.CanBeReverted() {
			perms.CanRevert = true
		}
		if v.GetSolicitud() != nil {
			perms.CanPrint = true
		}
	}

	// 2. Tomar / Asignar Cupo (Para el Target Suplente)
	hasTitular := targetUser.TitularID != nil
	if isDisponible && (isViewerAdminOrResponsable || !isVencido) && hasTitular && v.SenAsignadoID == *targetUser.TitularID {
		// Opción 1: El mismo suplente toma su cupo (Solo si no tiene cupo)
		if isTargetSuplente && !targetHasQuota && isViewerSuplente && authUser.ID == targetUser.ID {
			perms.CanTomarCupo = true
		}

		// Opción 2: Encargado asigna (Solo si no tiene cupo)
		if isEncargado && !targetHasQuota && isTargetSuplente {
			perms.CanAsignarCupo = true
		}

		// Opción 3: Admin/Responsable asigna (Solo si no tiene cupo)
		if isViewerAdminOrResponsable && !targetHasQuota && isTargetSuplente && !isTransferido {
			perms.CanAsignarCupo = true
		}
	}

	// 3. Transferencia (Admin/Responsable)
	if isViewerAdminOrResponsable && !isTransferido && !perms.CanAsignarCupo {
		perms.CanTransfer = true
	}

	// 4. Solicitudes (Owner, Admin or Encargado)
	if isOwner || isViewerAdminOrResponsable || isEncargado {
		sol := v.GetSolicitud()
		hasOrigin := targetUser.OrigenIATA != nil && *targetUser.OrigenIATA != ""

		// Creation
		if sol == nil {
			if hasOrigin && (isViewerAdminOrResponsable || (!isVencido && (isOwner || isEncargado))) {
				perms.CanCreate = true
				perms.CanCreateIdaVuelta = true
			}
		} else {
			// Edit/View permissions
			if sol.CanEdit(authUser) {
				perms.CanEdit = true
			}
			if sol.CanView(authUser) {
				perms.CanView = true
				perms.CanPrint = true
			}
		}
	}

	return perms
}

func (v CupoDerechoItem) GetWeekLabel() string {
	meses := []string{"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}
	mesStr := ""
	if v.Mes >= 1 && v.Mes <= 12 {
		mesStr = meses[v.Mes]
	}
	return fmt.Sprintf("%s/%s %d, %s", v.Semana, mesStr, v.Gestion, v.GetRangeLabel())
}

func (v CupoDerechoItem) GetRangeLabel() string {
	if v.FechaDesde == nil || v.FechaHasta == nil {
		return "Sin fechas"
	}

	dias := map[time.Weekday]string{
		time.Monday: "lun", time.Tuesday: "mar", time.Wednesday: "mie",
		time.Thursday: "jue", time.Friday: "vie", time.Saturday: "sab", time.Sunday: "dom",
	}
	meses := map[time.Month]string{
		time.January: "ene", time.February: "feb", time.March: "mar",
		time.April: "abr", time.May: "may", time.June: "jun",
		time.July: "jul", time.August: "ago", time.September: "sep",
		time.October: "oct", time.November: "nov", time.December: "dic",
	}

	d1, m1 := dias[v.FechaDesde.Weekday()], meses[v.FechaDesde.Month()]
	d2, m2 := dias[v.FechaHasta.Weekday()], meses[v.FechaHasta.Month()]

	return fmt.Sprintf("(%s %02d/%s  -  %s %02d/%s)",
		d1, v.FechaDesde.Day(), m1,
		d2, v.FechaHasta.Day(), m2)
}
