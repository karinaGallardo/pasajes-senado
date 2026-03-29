package models

import "time"

type Pasaje struct {
	BaseModel
	SolicitudID string `gorm:"not null;size:36"`

	SolicitudItemID *string        `gorm:"size:36;index"`
	SolicitudItem   *SolicitudItem `gorm:"foreignKey:SolicitudItemID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	AerolineaID *string    `gorm:"size:36"`
	Aerolinea   *Aerolinea `gorm:"foreignKey:AerolineaID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	AgenciaID *string  `gorm:"size:36"`
	Agencia   *Agencia `gorm:"foreignKey:AgenciaID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	NumeroVuelo string `gorm:"size:50"`
	Ruta        string `gorm:"size:255"`

	FechaVuelo   time.Time  `gorm:"type:timestamp"`
	FechaEmision *time.Time `gorm:"type:date"`

	CodigoReserva string  `gorm:"size:50"`
	NumeroBoleto  string  `gorm:"size:100;index"`
	Costo         float64 `gorm:"type:decimal(10,2)"`

	EstadoPasajeCodigo *string       `gorm:"size:50;default:'EMITIDO'"`
	EstadoPasaje       *EstadoPasaje `gorm:"foreignKey:EstadoPasajeCodigo;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	Archivo string `gorm:"size:255;default:''"`

	ArchivoPaseAbordo string  `gorm:"size:255;default:''"`
	PasajeAnteriorID  *string `gorm:"size:36"`
	PasajeAnterior    *Pasaje `gorm:"foreignKey:PasajeAnteriorID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`
	Glosa             string  `gorm:"type:text"`
	NumeroFactura     string  `gorm:"size:50;index"`
	CostoPenalidad    float64 `gorm:"type:decimal(10,2);default:0"`
}

func (Pasaje) TableName() string {
	return "pasajes"
}

func (p Pasaje) GetEstado() string {
	if p.EstadoPasaje != nil {
		return p.EstadoPasaje.Codigo
	}
	if p.EstadoPasajeCodigo == nil {
		return "EMITIDO"
	}
	return *p.EstadoPasajeCodigo
}

func (p Pasaje) GetEstadoCodigo() string {
	if p.EstadoPasajeCodigo == nil {
		return ""
	}
	return *p.EstadoPasajeCodigo
}

func (p Pasaje) CanBeEdited(user *Usuario) bool {
	if user == nil {
		return false
	}
	return user.IsAdminOrResponsable() && p.GetEstado() == "REGISTRADO"
}

func (p Pasaje) CanBeEmitted(user *Usuario) bool {
	if user == nil {
		return false
	}
	return user.IsAdminOrResponsable() && p.GetEstado() == "REGISTRADO"
}

func (p Pasaje) CanBeReverted(user *Usuario) bool {
	if user == nil {
		return false
	}
	return user.IsAdminOrResponsable() && p.GetEstado() == "EMITIDO"
}

func (p Pasaje) CanBeAnulado(user *Usuario) bool {
	if user == nil {
		return false
	}
	return user.IsAdminOrResponsable() && p.GetEstado() == "REGISTRADO"
}

func (p Pasaje) CanMarkUsado(user *Usuario) bool {
	st := p.GetEstadoCodigo()
	return (st == "EMITIDO" || st == "USADO")
}

func (p Pasaje) GetStatusBannerClass() string {
	switch p.GetEstado() {
	case "EMITIDO":
		return "bg-success-600"
	case "ANULADO":
		return "bg-neutral-600"
	case "USADO":
		return "bg-primary-600"
	default:
		return "bg-secondary-600"
	}
}
