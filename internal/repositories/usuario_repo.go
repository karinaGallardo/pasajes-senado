package repositories

import (
	"context"
	"sistema-pasajes/internal/models"
	"strings"

	"gorm.io/gorm"
)

type UsuarioRepository struct {
	db *gorm.DB
}

type PaginatedUsers struct {
	Usuarios   []models.Usuario
	Total      int64
	Page       int
	Limit      int
	TotalPages int
	SearchTerm string
}

func NewUsuarioRepository(db *gorm.DB) *UsuarioRepository {
	return &UsuarioRepository{db: db}
}

func (r *UsuarioRepository) WithTx(tx *gorm.DB) *UsuarioRepository {
	return &UsuarioRepository{db: tx}
}

func (r *UsuarioRepository) WithContext(ctx context.Context) *UsuarioRepository {
	return &UsuarioRepository{db: r.db.WithContext(ctx)}
}

func (r *UsuarioRepository) FindAll(ctx context.Context) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.WithContext(ctx).Preload("Rol").Preload("Genero").Order("created_at desc").Find(&usuarios).Error
	return usuarios, err
}

func FilterByRoleType(roleType string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch roleType {
		case "SENADOR":
			return db.Where("tipo = ?", "SENADOR_TITULAR")
		case "FUNCIONARIO":
			return db.Where("tipo IN ? OR rol_codigo IN ?",
				[]string{"FUNCIONARIO", "FUNCIONARIO_PERMANENTE", "FUNCIONARIO_EVENTUAL"},
				[]string{"ADMIN", "TECNICO", "USUARIO", "FUNCIONARIO", "RESPONSABLE"})
		default:
			return db
		}
	}
}

func SearchUsuario(term string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if term == "" {
			return db
		}

		words := strings.Fields(term)
		for _, word := range words {
			likeTerm := "%" + word + "%"
			db = db.Where("(username ILIKE ? OR ci ILIKE ? OR firstname ILIKE ? OR lastname ILIKE ? OR email ILIKE ?)",
				likeTerm, likeTerm, likeTerm, likeTerm, likeTerm)
		}

		return db
	}
}

func (r *UsuarioRepository) FindPaginated(ctx context.Context, roleType string, page, limit int, searchTerm string) (*PaginatedUsers, error) {
	var usuarios []models.Usuario
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&models.Usuario{}).
		Preload("Rol").
		Preload("Genero").
		Preload("Origen").
		Preload("Departamento").
		Preload("Cargo").
		Preload("Oficina").
		Preload("Titular").
		Preload("Suplentes").
		Scopes(FilterByRoleType(roleType), SearchUsuario(searchTerm))
	baseQuery.Count(&total)

	err := baseQuery.
		Scopes(Paginate(page, limit)).
		Order("lastname ASC, firstname ASC").
		Find(&usuarios).Error

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return &PaginatedUsers{
		Usuarios:   usuarios,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		SearchTerm: searchTerm,
	}, err
}

func (r *UsuarioRepository) FindByRoleType(ctx context.Context, roleType string) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	query := r.db.WithContext(ctx).Preload("Rol").Preload("Genero").Preload("Origen").Preload("Departamento")

	switch roleType {
	case "SENADOR":
		query = query.Preload("Titular").
			Preload("Cargo").Preload("Oficina").
			Preload("Suplentes").Preload("Suplentes.Origen").Preload("Suplentes.Departamento").
			Preload("Suplentes.Cargo").Preload("Suplentes.Oficina").
			Where("tipo = ?", "SENADOR_TITULAR").
			Order("lastname ASC, firstname ASC")
	case "FUNCIONARIO":
		query = query.Preload("Cargo").Preload("Oficina").
			Where("tipo IN ? OR rol_codigo IN ?",
				[]string{"FUNCIONARIO", "FUNCIONARIO_PERMANENTE", "FUNCIONARIO_EVENTUAL"},
				[]string{"ADMIN", "TECNICO", "USUARIO", "FUNCIONARIO", "RESPONSABLE"}).
			Order("lastname ASC, firstname ASC")
	default:
		query = query.Order("created_at desc")
	}

	err := query.Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) FindByID(ctx context.Context, id string) (*models.Usuario, error) {
	var usuario models.Usuario
	err := r.db.WithContext(ctx).Preload("Rol").
		Preload("Genero").
		Preload("Encargado").
		Preload("Origen").
		Preload("Departamento").
		First(&usuario, "id = ?", id).Error
	return &usuario, err
}

func (r *UsuarioRepository) FindByIDs(ctx context.Context, ids []string) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) UpdateRol(ctx context.Context, id string, rolCodigo string) error {
	return r.db.WithContext(ctx).Model(&models.Usuario{}).Where("id = ?", id).Update("rol_codigo", rolCodigo).Error
}

func (r *UsuarioRepository) Update(ctx context.Context, usuario *models.Usuario) error {
	return r.db.WithContext(ctx).Save(usuario).Error
}

func (r *UsuarioRepository) FindByCI(ctx context.Context, ci string) (*models.Usuario, error) {
	var usuario models.Usuario
	err := r.db.WithContext(ctx).Preload("Rol").Where("ci = ?", ci).First(&usuario).Error
	return &usuario, err
}

func (r *UsuarioRepository) FindByUsername(ctx context.Context, username string) (*models.Usuario, error) {
	var user models.Usuario
	err := r.db.WithContext(ctx).Preload("Rol").Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *UsuarioRepository) FindByCIUnscoped(ctx context.Context, ci string) (*models.Usuario, error) {
	var usuario models.Usuario
	err := r.db.WithContext(ctx).Unscoped().Preload("Rol").Where("ci = ?", ci).First(&usuario).Error
	return &usuario, err
}

func (r *UsuarioRepository) FindByUsernameUnscoped(ctx context.Context, username string) (*models.Usuario, error) {
	var user models.Usuario
	err := r.db.WithContext(ctx).
		Unscoped().
		Preload("Rol").
		Where("username = ?", username).
		First(&user).
		Error
	return &user, err
}

func (r *UsuarioRepository) Save(ctx context.Context, usuario *models.Usuario) error {
	return r.db.WithContext(ctx).Save(usuario).Error
}

func (r *UsuarioRepository) Refresh(ctx context.Context, usuario *models.Usuario) error {
	return r.db.WithContext(ctx).Preload("Rol").First(usuario).Error
}

func (r *UsuarioRepository) FindByEncargadoID(ctx context.Context, encargadoID string) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.WithContext(ctx).
		Preload("Rol").
		Preload("Genero").
		Preload("Origen").
		Preload("Cargo").
		Where("encargado_id = ?", encargadoID).
		Find(&usuarios).
		Error
	return usuarios, err
}

func (r *UsuarioRepository) FindSuplenteByTitularID(ctx context.Context, titularID string) (*models.Usuario, error) {
	var usuario models.Usuario
	err := r.db.WithContext(ctx).Preload("Rol").Preload("Genero").Where("titular_id = ?", titularID).First(&usuario).Error
	return &usuario, err
}

func (r *UsuarioRepository) Delete(ctx context.Context, usuario *models.Usuario) error {
	return r.db.WithContext(ctx).Delete(usuario).Error
}

func (r *UsuarioRepository) Restore(ctx context.Context, usuario *models.Usuario) error {
	return r.db.WithContext(ctx).Model(usuario).Unscoped().Update("deleted_at", nil).Error
}

func (r *UsuarioRepository) FindAllSenators(ctx context.Context) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.WithContext(ctx).Where("tipo IN ?", []string{"SENADOR_TITULAR", "SENADOR_SUPLENTE"}).Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) FindAdminsAndResponsables(ctx context.Context) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.WithContext(ctx).Where("rol_codigo IN ?", []string{"ADMIN", "RESPONSABLE"}).
		Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) RunTransaction(fn func(repo *UsuarioRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		return fn(txRepo)
	})
}
