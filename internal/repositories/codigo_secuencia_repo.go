package repositories

import (
	"context"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CodigoSecuenciaRepository struct {
	db *gorm.DB
}

func NewCodigoSecuenciaRepository() *CodigoSecuenciaRepository {
	return &CodigoSecuenciaRepository{db: configs.DB}
}

func (r *CodigoSecuenciaRepository) WithTx(tx *gorm.DB) *CodigoSecuenciaRepository {
	return &CodigoSecuenciaRepository{db: tx}
}

func (r *CodigoSecuenciaRepository) WithContext(ctx context.Context) *CodigoSecuenciaRepository {
	return &CodigoSecuenciaRepository{db: r.db.WithContext(ctx)}
}

func (r *CodigoSecuenciaRepository) GetNext(gestion int, tipo string) (int, error) {
	var nextVal int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var c models.CodigoSecuencia
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("gestion = ? AND tipo = ?", gestion, tipo).First(&c).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c = models.CodigoSecuencia{
					Gestion: gestion,
					Tipo:    tipo,
					Numero:  0,
				}
				if err := tx.Create(&c).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		c.Numero++
		if err := tx.Save(&c).Error; err != nil {
			return err
		}
		nextVal = c.Numero
		return nil
	})
	return nextVal, err
}
