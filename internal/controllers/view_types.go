package controllers

import (
	"sistema-pasajes/internal/models"
)

type PasajeView struct {
	models.Pasaje
	Perms            models.PasajePermissions
	StatusColorClass string
}
