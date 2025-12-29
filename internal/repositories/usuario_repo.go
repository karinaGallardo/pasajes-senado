package repositories

import (
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

type UsuarioRepository struct{}

func NewUsuarioRepository() *UsuarioRepository {
	return &UsuarioRepository{}
}

func (r *UsuarioRepository) FindAll() ([]models.Usuario, error) {
	var usuarios []models.Usuario
	err := configs.DB.Preload("Rol").Preload("Genero").Order("created_at desc").Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) FindByRoleType(roleType string) ([]models.Usuario, error) {
	var usuarios []models.Usuario
	query := configs.DB.Preload("Rol").Preload("Genero").Order("created_at desc")

	switch roleType {
	case "SENADOR":
		query = query.Where("tipo IN ?", []string{"SENADOR_TITULAR", "SENADOR_SUPLENTE"})
	case "FUNCIONARIO":
		query = query.Where("tipo IN ?", []string{"FUNCIONARIO", "FUNCIONARIO_PERMANENTE", "FUNCIONARIO_EVENTUAL"})
	}

	err := query.Find(&usuarios).Error
	return usuarios, err
}

func (r *UsuarioRepository) FindByID(id string) (*models.Usuario, error) {
	var usuario models.Usuario
	err := configs.DB.Preload("Rol").Preload("Genero").Preload("Encargado").First(&usuario, "id = ?", id).Error
	return &usuario, err
}

func (r *UsuarioRepository) UpdateRol(id string, rolCodigo string) error {
	return configs.DB.Model(&models.Usuario{}).Where("id = ?", id).Update("rol_id", rolCodigo).Error
}

func (r *UsuarioRepository) Update(usuario *models.Usuario) error {
	return configs.DB.Save(usuario).Error
}
