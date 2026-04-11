package repositories

import (
	"context"
	"sistema-pasajes/internal/models"
	"strings"

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

type PaginatedRutas struct {
	Rutas      []models.Ruta
	Total      int64
	Page       int
	Limit      int
	TotalPages int
	SearchTerm string
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

func (r *RutaRepository) FindPaginated(ctx context.Context, page, limit int, query string) (*PaginatedRutas, error) {
	var rutas []models.Ruta
	var total int64

	db := r.db.WithContext(ctx).Model(&models.Ruta{}).
		Preload("Origen").
		Preload("Destino").
		Preload("Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Escalas.Destino").
		Preload("Contratos.Aerolinea")

	if query != "" {
		words := strings.Fields(strings.ToLower(query))
		for _, word := range words {
			w := "%" + word + "%"
			db = db.Where("(tramo ILIKE ? OR sigla ILIKE ? OR origen_iata ILIKE ? OR destino_iata ILIKE ?)", w, w, w, w)
		}
	}

	db.Count(&total)

	err := db.Scopes(Paginate(page, limit)).
		Order("tramo ASC").
		Find(&rutas).Error

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return &PaginatedRutas{
		Rutas:      rutas,
		Total:       total,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		SearchTerm:  query,
	}, err
}

func (r *RutaRepository) Search(ctx context.Context, query string, onlyAtomic bool) ([]models.Ruta, error) {
	var rutas []models.Ruta
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

	// Multi-keyword matching logic
	words := strings.Fields(strings.ToLower(query))
	for _, word := range words {
		w := "%" + word + "%"
		db = db.Where("(tramo ILIKE ? OR sigla ILIKE ? OR origen_iata ILIKE ? OR destino_iata ILIKE ?)", w, w, w, w)
	}

	err := db.Limit(100).Find(&rutas).Error
	return rutas, err
}

func (r *RutaRepository) Create(ctx context.Context, ruta *models.Ruta) error {
	return r.db.WithContext(ctx).Create(ruta).Error
}

func (r *RutaRepository) FindByID(ctx context.Context, id string) (*models.Ruta, error) {
	var ruta models.Ruta
	err := r.db.WithContext(ctx).
		Preload("Origen").
		Preload("Destino").
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

func (r *RutaRepository) FindDestinosByIATAs(ctx context.Context, iatas []string) ([]models.Destino, error) {
	var destinos []models.Destino
	err := r.db.WithContext(ctx).Where("iata IN (?)", iatas).Find(&destinos).Error
	return destinos, err
}
