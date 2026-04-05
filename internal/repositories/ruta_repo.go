package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type RutaRepository struct {
	db *gorm.DB
}

func NewRutaRepository(db *gorm.DB) *RutaRepository {
	return &RutaRepository{db: db}
}

func (r *RutaRepository) WithContext(ctx context.Context) *RutaRepository {
	return &RutaRepository{db: r.db.WithContext(ctx)}
}

func (r *RutaRepository) FindAll(ctx context.Context) ([]models.Ruta, error) {
	var rutas []models.Ruta
	err := r.db.WithContext(ctx).
		Preload("Origen").
		Preload("Destino").
		Preload("Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Escalas.Destino").
		Preload("Contratos.Aerolinea").
		Find(&rutas).Error
	return rutas, err
}

func (r *RutaRepository) Search(ctx context.Context, query string, onlyAtomic bool) ([]models.Ruta, error) {
	var rutas []models.Ruta
	q := "%" + query + "%"
	db := r.db.WithContext(ctx).
		Preload("Origen").
		Preload("Destino")

	if onlyAtomic {
		db = db.Where("rutas.id NOT IN (SELECT ruta_id FROM ruta_escalas)")
	} else {
		db = db.Preload("Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).Preload("Escalas.Destino")
	}

	err := db.Where("tramo ILIKE ? OR sigla ILIKE ? OR origen_iata ILIKE ? OR destino_iata ILIKE ?", q, q, q, q).
		Limit(20).
		Find(&rutas).Error
	return rutas, err
}

func (r *RutaRepository) Create(ctx context.Context, ruta *models.Ruta) error {
	return r.db.WithContext(ctx).Create(ruta).Error
}

func (r *RutaRepository) FindByID(ctx context.Context, id string) (*models.Ruta, error) {
	var ruta models.Ruta
	err := r.db.WithContext(ctx).
		Preload("Contratos.Aerolinea").
		Preload("Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Escalas.Destino").First(&ruta, "id = ?", id).Error
	return &ruta, err
}

func (r *RutaRepository) AssignContract(ctx context.Context, contrato *models.RutaContrato) error {
	var existing models.RutaContrato
	err := r.db.WithContext(ctx).Where("ruta_id = ? AND aerolinea_id = ?", contrato.RutaID, contrato.AerolineaID).First(&existing).Error
	if err == nil {
		// Update existing
		return r.db.WithContext(ctx).Model(&existing).Update("monto_referencial", contrato.MontoReferencial).Error
	}
	return r.db.WithContext(ctx).Create(contrato).Error
}

func (r *RutaRepository) GetContractsByRuta(ctx context.Context, rutaID string) ([]models.RutaContrato, error) {
	var contratos []models.RutaContrato
	err := r.db.WithContext(ctx).Preload("Aerolinea").Where("ruta_id = ?", rutaID).Find(&contratos).Error
	return contratos, err
}

func (r *RutaRepository) DeleteContract(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.RutaContrato{}, "id = ?", id).Error
}
