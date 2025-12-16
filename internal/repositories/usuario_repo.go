package repositories

import (
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

type UsuarioRepository struct{}

func NewUsuarioRepository() *UsuarioRepository {
	return &UsuarioRepository{}
}

func (r *UsuarioRepository) FindByUsername(username string) (*models.Usuario, error) {
	var user models.Usuario
	err := configs.DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
