package models

type CodigoSecuencia struct {
	BaseModel
	Gestion int    `gorm:"index:idx_gestion_tipo,unique"`
	Tipo    string `gorm:"size:20;index:idx_gestion_tipo,unique"`
	Numero  int    `gorm:"default:0"`
}

func (CodigoSecuencia) TableName() string {
	return "codigo_secuencias"
}
