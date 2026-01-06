package repositories

import (
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

func (r *UsuarioRepository) FindAll() ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.Preload("Rol").Preload("Genero").Order("created_at desc").Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) FindPaginated(roleType string, page, limit int, searchTerm string) (*PaginatedUsers, error) {
	var usuarios []models.Usuario
	var total int64

	query := r.db.Model(&models.Usuario{}).Preload("Rol").Preload("Genero").Preload("Origen").Preload("Departamento").Preload("Cargo")

	switch roleType {
	case "SENADOR":
		query = query.Where("tipo = ?", "SENADOR_TITULAR")
	case "FUNCIONARIO":
		query = query.Where("tipo IN ?", []string{"FUNCIONARIO", "FUNCIONARIO_PERMANENTE", "FUNCIONARIO_EVENTUAL"})
	}

	if searchTerm != "" {
		likeTerm := "%" + searchTerm + "%"
		query = query.Where("(username LIKE ? OR ci LIKE ? OR firstname LIKE ? OR lastname LIKE ? OR email LIKE ?)",
			likeTerm, likeTerm, likeTerm, likeTerm, likeTerm)
	}

	query.Count(&total)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	offset := (page - 1) * limit

	err := query.Order("lastname ASC, firstname ASC").
		Offset(offset).
		Limit(limit).
		Find(&usuarios).Error

	totalPages := int((total + int64(limit) - 1) / int64(limit))

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
			Where("tipo = ?", "SENADOR_TITULAR").
			Order("lastname ASC, firstname ASC")
	case "FUNCIONARIO":
		query = query.Preload("Cargo").
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
