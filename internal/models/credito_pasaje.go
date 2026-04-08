package models

import (
	"time"

	"gorm.io/gorm"
)

type EstadoCredito string

const (
	EstadoCreditoPendiente  EstadoCredito = "PENDIENTE"
	EstadoCreditoDisponible EstadoCredito = "DISPONIBLE"
	EstadoCreditoUsado      EstadoCredito = "USADO"
	EstadoCreditoVencido    EstadoCredito = "VENCIDO"
	EstadoCreditoCancelado  EstadoCredito = "CANCELADO"
)

// CreditoPasaje representa un "vale" o crédito de viaje generado por un tramo no utilizado
// en un descargo de derecho (senadores).
type CreditoPasaje struct {
	ID string `gorm:"primaryKey;type:uuid;default:uuidv7()" json:"id"`

	UsuarioID string   `gorm:"type:uuid;not null;index" json:"usuario_id"`
	Usuario   *Usuario `gorm:"foreignKey:UsuarioID" json:"usuario,omitempty"`

	DescargoID string    `gorm:"type:uuid;not null;index" json:"descargo_id"`
	Descargo   *Descargo `gorm:"foreignKey:DescargoID" json:"descargo,omitempty"`

	Monto              float64       `gorm:"type:decimal(10,2);not null" json:"monto"`
	RutaReferencia     string        `gorm:"type:text" json:"ruta_referencia"`     // Ej: Pando-Beni, Beni-VVI, VVI-LPB
	BilletesReferencia string        `gorm:"type:text" json:"billetes_referencia"` // Números de billete involucrados
	Estado             EstadoCredito `gorm:"type:varchar(20);default:'PENDIENTE';index" json:"estado"`

	// Si se usa en una solicitud, guardamos la referencia
	SolicitudUsoID *string    `gorm:"type:uuid;index" json:"solicitud_uso_id"`
	FechaUso       *time.Time `json:"fecha_uso"`

	Observaciones string `gorm:"type:text" json:"observaciones"`

	CreatedAt time.Time      `gorm:"index;type:timestamp"`
	UpdatedAt time.Time      `gorm:"type:timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"index;type:timestamp"`

	CreatedBy *string `gorm:"size:36;default:null"`
	UpdatedBy *string `gorm:"size:36;default:null"`
	DeletedBy *string `gorm:"size:36;default:null"`
}

func (c CreditoPasaje) IsDisponible() bool {
	return c.Estado == EstadoCreditoDisponible
}
