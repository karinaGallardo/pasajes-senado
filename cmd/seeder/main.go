package main

import (
	"fmt"
	"log"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

func main() {
	configs.ConnectDB()

	ciudadRepo := repositories.NewCiudadRepository()
	if err := ciudadRepo.SeedDefaults(); err != nil {
		log.Printf("Error seeding ciudades: %v", err)
	} else {
		fmt.Println("Ciudades inicializadas.")
	}

	seedRolesAndPermissions()
}

func seedRolesAndPermissions() {
	roles := []models.Rol{
		{Codigo: "ADMIN", Nombre: "Administrador del Sistema"},
		{Codigo: "TECNICO", Nombre: "Técnico de Sistema"},
		{Codigo: "SENADOR", Nombre: "Honorable Senador"},
		{Codigo: "FUNCIONARIO", Nombre: "Funcionario Administrativo"},
	}

	fmt.Println("Sincronizando Roles...")
	roleMap := make(map[string]*models.Rol)
	for _, r := range roles {
		var role models.Rol
		if err := configs.DB.Where("codigo = ?", r.Codigo).FirstOrCreate(&role, r).Error; err != nil {
			log.Fatalf("Error creando rol %s: %v", r.Codigo, err)
		}
		roleMap[r.Codigo] = &role
	}

	permisos := []models.Permiso{
		{Codigo: "solicitud:crear", Nombre: "Crear Solicitud", Descripcion: "Permite crear nuevas solicitudes de pasajes"},
		{Codigo: "solicitud:ver_propias", Nombre: "Ver Mis Solicitudes", Descripcion: "Permite ver solicitudes propias"},
		{Codigo: "solicitud:ver_todas", Nombre: "Ver Todas Solicitudes", Descripcion: "Permite ver todas las solicitudes (Admin/Operador)"},
		{Codigo: "solicitud:aprobar", Nombre: "Aprobar Solicitud", Descripcion: "Permite aprobar o rechazar solicitudes"},

		{Codigo: "descargo:crear", Nombre: "Crear Descargo", Descripcion: "Permite subir descargos de pasajes"},
		{Codigo: "descargo:ver", Nombre: "Ver Descargos", Descripcion: "Permite ver descargos"},

		{Codigo: "usuario:ver", Nombre: "Ver Usuarios", Descripcion: "Permite listar usuarios"},
		{Codigo: "usuario:editar", Nombre: "Editar Usuarios", Descripcion: "Permite editar roles de usuarios"},
		{Codigo: "reporte:ver", Nombre: "Ver Reportes", Descripcion: "Permite ver reportes globales"},
	}

	fmt.Println("Sincronizando Permisos...")
	permisoMap := make(map[string]*models.Permiso)
	for _, p := range permisos {
		var perm models.Permiso
		if err := configs.DB.Where("codigo = ?", p.Codigo).FirstOrCreate(&perm, p).Error; err != nil {
			log.Fatalf("Error creando permiso %s: %v", p.Codigo, err)
		}
		permisoMap[p.Codigo] = &perm
	}

	rolBasicoPermisos := []*models.Permiso{
		permisoMap["solicitud:crear"],
		permisoMap["solicitud:ver_propias"],
		permisoMap["descargo:crear"],
		permisoMap["descargo:ver"],
	}

	rolTecnicoPermisos := []*models.Permiso{
		permisoMap["solicitud:crear"],
		permisoMap["solicitud:ver_propias"],
		permisoMap["solicitud:ver_todas"],
		permisoMap["descargo:crear"],
		permisoMap["descargo:ver"],
		permisoMap["usuario:ver"],
	}

	var allPermisos []*models.Permiso
	for _, p := range permisoMap {
		allPermisos = append(allPermisos, p)
	}

	assignPermsToRole(roleMap["FUNCIONARIO"], rolBasicoPermisos)
	assignPermsToRole(roleMap["SENADOR"], rolBasicoPermisos)
	assignPermsToRole(roleMap["TECNICO"], rolTecnicoPermisos)
	assignPermsToRole(roleMap["ADMIN"], allPermisos)

	fmt.Println("Roles y Permisos sincronizados correctamente.")
}

func assignPermsToRole(rol *models.Rol, perms []*models.Permiso) {
	if rol == nil {
		return
	}
	configs.DB.Model(rol).Association("Permisos").Replace(perms)
}
