package models

import "time"

type UsuarioOrigenAlternativo struct {
	ID        string    `gorm:"primaryKey;type:varchar(36);default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time `gorm:"type:timestamp"`
	UpdatedAt time.Time `gorm:"type:timestamp"`

	UsuarioID   string   `gorm:"size:36;not null;uniqueIndex:idx_user_iata"`
	DestinoIATA string   `gorm:"column:iata;size:5;not null;uniqueIndex:idx_user_iata"`
	Destino     *Destino `gorm:"foreignKey:DestinoIATA;references:IATA;<-:false"`
}

func (UsuarioOrigenAlternativo) TableName() string {
	return "usuarios_origenes_alternativos"
}
