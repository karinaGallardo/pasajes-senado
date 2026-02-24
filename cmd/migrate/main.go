package main

import (
	"log"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

func main() {
	configs.ConnectDB()
	log.Println("Iniciando migración completa de base de datos...")

	err := configs.DB.AutoMigrate(
		// Seguridad y Usuarios
		&models.Permiso{},
		&models.Rol{},
		&models.Usuario{},
		&models.Genero{},

		// Organigrama
		&models.Cargo{},
		&models.Oficina{},

		// Catálogos Geográficos y de Transporte
		&models.Destino{},
		// &models.Ciudad{},
		&models.Departamento{},
		&models.Aerolinea{},
		&models.Agencia{},
		&models.Ruta{},
		&models.RutaEscala{},
		&models.RutaContrato{},

		// Configuración del Sistema
		&models.Configuracion{},
		&models.CodigoSecuencia{},
		&models.CategoriaViatico{},
		&models.CategoriaCompensacion{},

		// Definiciones de Viaje
		&models.ConceptoViaje{},
		&models.TipoSolicitud{},
		&models.AmbitoViaje{},
		&models.TipoItinerario{},
		&models.EstadoSolicitud{},
		&models.EstadoSolicitudItem{},
		&models.EstadoPasaje{},

		// Gestión de Cupos
		&models.EstadoCupoDerecho{},
		&models.CupoDerecho{},
		&models.CupoDerechoItem{},

		// Operaciones Principales
		&models.Solicitud{},
		&models.SolicitudItem{},
		&models.Pasaje{},
		&models.Viatico{},
		&models.DetalleViatico{},
		&models.ZonaViatico{},
		&models.CategoriaViatico{},
		&models.Agencia{},
		&models.Compensacion{},
		&models.Descargo{},
		&models.DescargoOficial{},
		&models.AnexoDescargo{},
		&models.DocumentoDescargo{},
		&models.DetalleItinerarioDescargo{},
	)

	if err != nil {
		log.Fatalf("Error durante la migración: %v", err)
	}

	log.Println("Migración completada exitosamente.")
}
