package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

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

func NewUsuarioRepository() *UsuarioRepository {
	return &UsuarioRepository{db: configs.DB}
}

func (r *UsuarioRepository) WithTx(tx *gorm.DB) *UsuarioRepository {
	return &UsuarioRepository{db: tx}
}

func (r *UsuarioRepository) WithContext(ctx context.Context) *UsuarioRepository {
	return &UsuarioRepository{db: r.db.WithContext(ctx)}
}

func (r *UsuarioRepository) FindAll() ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.Preload("Rol").Preload("Genero").Order("created_at desc").Find(&usuarios).Error
	return usuarios, err
}

func FilterByRoleType(roleType string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch roleType {
		case "SENADOR":
			return db.Where("tipo = ?", "SENADOR_TITULAR")
		case "FUNCIONARIO":
			return db.Where("tipo IN ?", []string{"FUNCIONARIO", "FUNCIONARIO_PERMANENTE", "FUNCIONARIO_EVENTUAL"})
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
		likeTerm := "%" + term + "%"
		return db.Where("(username ILIKE ? OR ci ILIKE ? OR firstname ILIKE ? OR lastname ILIKE ? OR email ILIKE ?)",
			likeTerm, likeTerm, likeTerm, likeTerm, likeTerm)
	}
}

func (r *UsuarioRepository) FindPaginated(roleType string, page, limit int, searchTerm string) (*PaginatedUsers, error) {
	var usuarios []models.Usuario
	var total int64

	baseQuery := r.db.Model(&models.Usuario{}).
		Preload("Rol").
		Preload("Genero").
		Preload("Origen").
		Preload("Departamento").
		Preload("Cargo").
		Preload("Oficina").
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

func (r *UsuarioRepository) FindByRoleType(roleType string) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	query := r.db.Preload("Rol").Preload("Genero").Preload("Origen").Preload("Departamento")

	switch roleType {
	case "SENADOR":
		query = query.Preload("Suplentes").Preload("Suplentes.Origen").Preload("Suplentes.Departamento").
			Where("tipo IN ?", []string{"SENADOR_TITULAR", "SENADOR_SUPLENTE"}).
			Order("lastname ASC, firstname ASC")
	case "FUNCIONARIO":
		query = query.Preload("Cargo").Preload("Oficina").
			Where("tipo IN ?", []string{"FUNCIONARIO", "FUNCIONARIO_PERMANENTE", "FUNCIONARIO_EVENTUAL"}).
			Order("lastname ASC, firstname ASC")
	default:
		query = query.Order("created_at desc")
	}

	err := query.Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) FindByID(id string) (*models.Usuario, error) {
	var usuario models.Usuario
	err := r.db.Preload("Rol").
		Preload("Genero").
		Preload("Encargado").
		Preload("Origen").
		Preload("Departamento").
		First(&usuario, "id = ?", id).Error
	return &usuario, err
}

func (r *UsuarioRepository) FindByIDs(ids []string) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.Where("id IN ?", ids).Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) UpdateRol(id string, rolCodigo string) error {
	return r.db.Model(&models.Usuario{}).Where("id = ?", id).Update("rol_codigo", rolCodigo).Error
}

func (r *UsuarioRepository) Update(usuario *models.Usuario) error {
	return r.db.Save(usuario).Error
}

func (r *UsuarioRepository) FindByCI(ci string) (*models.Usuario, error) {
	var usuario models.Usuario
	err := r.db.Preload("Rol").Where("ci = ?", ci).First(&usuario).Error
	return &usuario, err
}

func (r *UsuarioRepository) FindByUsername(username string) (*models.Usuario, error) {
	var user models.Usuario
	err := r.db.Preload("Rol").Where("username = ?", username).First(&user).Error
	return &user, err
}

func (r *UsuarioRepository) FindByCIUnscoped(ci string) (*models.Usuario, error) {
	var usuario models.Usuario
	err := r.db.Unscoped().Preload("Rol").Where("ci = ?", ci).First(&usuario).Error
	return &usuario, err
}

func (r *UsuarioRepository) Save(usuario *models.Usuario) error {
	return r.db.Save(usuario).Error
}

func (r *UsuarioRepository) Refresh(usuario *models.Usuario) error {
	return r.db.Preload("Rol").First(usuario).Error
}

func (r *UsuarioRepository) FindByEncargadoID(encargadoID string) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.Preload("Rol").Preload("Genero").Preload("Origen").Preload("Cargo").Where("encargado_id = ?", encargadoID).Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) FindSuplenteByTitularID(titularID string) (*models.Usuario, error) {
	var usuario models.Usuario
	err := r.db.Preload("Rol").Preload("Genero").Where("titular_id = ?", titularID).First(&usuario).Error
	return &usuario, err
}

func (r *UsuarioRepository) Delete(usuario *models.Usuario) error {
	return r.db.Delete(usuario).Error
}

func (r *UsuarioRepository) Restore(usuario *models.Usuario) error {
	return r.db.Model(usuario).Unscoped().Update("deleted_at", nil).Error
}

func (r *UsuarioRepository) FindAllSenators() ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.Where("tipo IN ?", []string{"SENADOR_TITULAR", "SENADOR_SUPLENTE"}).Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) RunTransaction(fn func(repo *UsuarioRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		return fn(txRepo)
	})
}
