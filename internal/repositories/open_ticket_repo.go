package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type OpenTicketRepository struct {
	db *gorm.DB
}

func NewOpenTicketRepository(db *gorm.DB) *OpenTicketRepository {
	return &OpenTicketRepository{db: db}
}

func (r *OpenTicketRepository) WithTx(tx *gorm.DB) *OpenTicketRepository {
	return &OpenTicketRepository{db: tx}
}

func (r *OpenTicketRepository) FindByDescargoID(ctx context.Context, descargoID string) ([]models.OpenTicket, error) {
	var tickets []models.OpenTicket
	err := r.db.WithContext(ctx).
		Preload("Descargo.Solicitud").
		Preload("Pasaje").
		Where("descargo_id = ?", descargoID).
		Find(&tickets).Error
	return tickets, err
}

func (r *OpenTicketRepository) Create(ctx context.Context, ticket *models.OpenTicket) error {
	return r.db.WithContext(ctx).Create(ticket).Error
}

func (r *OpenTicketRepository) FindDisponiblesByUsuarioID(ctx context.Context, usuarioID string) ([]models.OpenTicket, error) {
	var tickets []models.OpenTicket
	err := r.db.WithContext(ctx).
		Preload("Descargo.Solicitud").
		Preload("Pasaje").
		Where("usuario_id = ? AND estado = ?", usuarioID, models.EstadoOpenTicketDisponible).
		Order("created_at DESC").
		Find(&tickets).Error
	return tickets, err
}

func (r *OpenTicketRepository) FindAllByUsuarioID(ctx context.Context, usuarioID string) ([]models.OpenTicket, error) {
	var tickets []models.OpenTicket
	err := r.db.WithContext(ctx).
		Preload("Descargo.Solicitud").
		Preload("Pasaje").
		Preload("Pasaje.Aerolinea").
		Preload("Pasaje.RutaPasaje").
		Where("usuario_id = ?", usuarioID).
		Order("created_at DESC").
		Find(&tickets).Error
	return tickets, err
}

func (r *OpenTicketRepository) FindByID(ctx context.Context, id string) (*models.OpenTicket, error) {
	var ticket models.OpenTicket
	err := r.db.WithContext(ctx).
		Preload("Usuario").
		Preload("Descargo.Solicitud").
		Preload("Pasaje").
		Preload("Pasaje.Aerolinea").
		Preload("Pasaje.RutaPasaje").
		First(&ticket, "id = ?", id).Error
	return &ticket, err
}

func (r *OpenTicketRepository) Update(ctx context.Context, ticket *models.OpenTicket) error {
	return r.db.WithContext(ctx).Save(ticket).Error
}

func (r *OpenTicketRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.OpenTicket{}, "id = ?", id).Error
}

func (r *OpenTicketRepository) FindAll(ctx context.Context, filters map[string]any) ([]models.OpenTicket, error) {
	var tickets []models.OpenTicket
	query := r.db.WithContext(ctx).
		Preload("Usuario").
		Preload("Descargo.Solicitud").
		Preload("Pasaje").
		Preload("Pasaje.Aerolinea").
		Preload("Pasaje.RutaPasaje").
		Order("created_at DESC")

	for k, v := range filters {
		if v != "" {
			query = query.Where(k+" = ?", v)
		}
	}

	err := query.Find(&tickets).Error
	return tickets, err
}

func (r *OpenTicketRepository) CountByEstado(ctx context.Context, estado models.EstadoOpenTicket, userIDs []string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.OpenTicket{}).Where("estado = ?", estado)
	if len(userIDs) > 0 {
		query = query.Where("usuario_id IN ?", userIDs)
	}
	err := query.Count(&count).Error
	return count, err
}
