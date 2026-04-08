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
		case models.RolSenador:
			return db.Where("tipo = ?", models.TipoSenadorTitular)
		case models.RolFuncionario:
			return db.Where("tipo IN ? OR rol_codigo IN ?",
				[]string{models.TipoFuncionario, models.TipoFuncionarioPermanente, models.TipoFuncionarioEventual},
				[]string{models.RolAdmin, models.RolTecnico, models.RolUsuario, models.RolFuncionario, models.RolResponsable})
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
			db = db.Where("(username ILIKE ? OR ci ILIKE ? OR firstname ILIKE ? OR secondname ILIKE ? OR lastname ILIKE ? OR surname ILIKE ? OR email ILIKE ?)",
				likeTerm, likeTerm, likeTerm, likeTerm, likeTerm, likeTerm, likeTerm)
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

func (r *UsuarioRepository) SearchStaff(ctx context.Context, query string) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.WithContext(ctx).
		Preload("Rol").
		Preload("Cargo").
		Preload("Oficina").
		Scopes(FilterByRoleType(models.RolFuncionario), SearchUsuario(query)).
		Order("lastname ASC, firstname ASC").
		Limit(20).
		Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) FindByRoleType(ctx context.Context, roleType string) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	query := r.db.WithContext(ctx).Preload("Rol").Preload("Genero").Preload("Origen").Preload("Departamento")

	switch roleType {
	case models.RolSenador:
		query = query.Preload("Titular").
			Preload("Cargo").Preload("Oficina").
			Preload("Suplentes").Preload("Suplentes.Origen").Preload("Suplentes.Departamento").
			Preload("Suplentes.Cargo").Preload("Suplentes.Oficina").
			Where("tipo = ?", models.TipoSenadorTitular).
			Order("lastname ASC, firstname ASC")
	case models.RolFuncionario:
		query = query.Preload("Cargo").Preload("Oficina").
			Where("tipo IN ? OR rol_codigo IN ?",
				[]string{models.TipoFuncionario, models.TipoFuncionarioPermanente, models.TipoFuncionarioEventual},
				[]string{models.RolAdmin, models.RolTecnico, models.RolUsuario, models.RolFuncionario, models.RolResponsable}).
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
		Preload("OrigenesAlternativos.Destino").
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

func (r *UsuarioRepository) SyncOrigenesAlternativos(ctx context.Context, usuarioID string, origins []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Eliminar existentes (Borrado FÍSICO para evitar conflictos con el índice único)
		if err := tx.Unscoped().Where("usuario_id = ?", usuarioID).Delete(&models.UsuarioOrigenAlternativo{}).Error; err != nil {
			return err
		}

		// Insertar nuevos (filtrando duplicados en Go antes de intentar insertarlos)
		uniqueOrigins := make(map[string]bool)
		for _, iata := range origins {
			if iata == "" || uniqueOrigins[iata] {
				continue
			}
			uniqueOrigins[iata] = true

			newOrigin := models.UsuarioOrigenAlternativo{
				UsuarioID:   usuarioID,
				DestinoIATA: iata,
			}
			if err := tx.Create(&newOrigin).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *UsuarioRepository) FindByCI(ctx context.Context, ci string) (*models.Usuario, error) {
	var usuario models.Usuario
	err := r.db.WithContext(ctx).
		Preload("Rol").
		Where("ci = ?", ci).
		First(&usuario).
		Error
	return &usuario, err
}

func (r *UsuarioRepository) FindByUsername(ctx context.Context, username string) (*models.Usuario, error) {
	var user models.Usuario
	err := r.db.WithContext(ctx).
		Preload("Rol").
		Where("username = ?", username).
		First(&user).
		Error
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
		Order("lastname ASC, firstname ASC").
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
	err := r.db.WithContext(ctx).
		Where("tipo IN ?", []string{models.TipoSenadorTitular, models.TipoSenadorSuplente}).
		Order("lastname ASC, firstname ASC").
		Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) FindAdminsAndResponsables(ctx context.Context) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := r.db.WithContext(ctx).Where("rol_codigo IN ?", []string{models.RolAdmin, models.RolResponsable}).
		Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) RunTransaction(fn func(repo *UsuarioRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		return fn(txRepo)
	})
}
