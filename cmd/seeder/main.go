package main

import (
	"fmt"
	"log"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

func main() {
	configs.ConnectDB()

	seedCiudades()
	seedDepartamentos()
	seedProveedores()
	seedRolesAndPermissions()
	seedCatalogosViaje()
	seedViaticosAndConfig()
}

func seedViaticosAndConfig() {
	fmt.Println("Sincronizando Categorías de Viáticos y Configuración...")

	categorias := []models.CategoriaViatico{
		{Nombre: "PRIMERA CATEGORIA", Codigo: 1, Monto: 359.00, Moneda: "Bs", Ubicacion: "INTERIOR"},
		{Nombre: "SEGUNDA CATEGORIA", Codigo: 2, Monto: 279.00, Moneda: "Bs", Ubicacion: "INTERIOR"},
		{Nombre: "TERCERA CATEGORIA", Codigo: 3, Monto: 212.00, Moneda: "Bs", Ubicacion: "INTERIOR"},
		{Nombre: "VIAJE AL EXTERIOR (ESCALA BASICA)", Codigo: 4, Monto: 300.00, Moneda: "USD", Ubicacion: "EXTERIOR"},
	}

	for _, c := range categorias {
		configs.DB.Where("nombre = ? AND ubicacion = ?", c.Nombre, c.Ubicacion).FirstOrCreate(&c)
	}

	confList := []models.Configuracion{
		{Clave: "RC_IVA_TASA", Valor: "0.13", Tipo: "FLOAT"},
		{Clave: "TC_USD_OFICIAL", Valor: "6.96", Tipo: "FLOAT"},
		{Clave: "GESTION_ACTUAL", Valor: "2025", Tipo: "INT"},
	}

	for _, cf := range confList {
		var existing models.Configuracion
		result := configs.DB.Where("clave = ?", cf.Clave).First(&existing)
		if result.Error != nil {
			configs.DB.Create(&cf)
		} else {
		}
	}
}

func seedProveedores() {
	fmt.Println("Sincronizando Proveedores (Aerolineas y Agencias)...")

	aerolineas := []models.Aerolinea{
		{Nombre: "BoA - Boliviana de Aviación", Estado: true},
		{Nombre: "EcoJet", Estado: true},
	}
	for _, a := range aerolineas {
		configs.DB.Where("nombre = ?", a.Nombre).FirstOrCreate(&a)
	}

	agencias := []models.Agencia{
		{Nombre: "Agencia de Viajes Cuarta Dimensión", Estado: true},
		{Nombre: "Tropical Tours", Estado: true},
		{Nombre: "Mundo Viajes", Estado: true},
	}
	for _, a := range agencias {
		configs.DB.Where("nombre = ?", a.Nombre).FirstOrCreate(&a)
	}
}

func seedCatalogosViaje() {
	fmt.Println("Sincronizando Catálogos de Viaje...")

	ambitoNac := models.AmbitoViaje{Codigo: "NACIONAL", Nombre: "Nacional"}
	ambitoInt := models.AmbitoViaje{Codigo: "INTERNACIONAL", Nombre: "Internacional"}
	configs.DB.FirstOrCreate(&ambitoNac, models.AmbitoViaje{Codigo: "NACIONAL"})
	configs.DB.FirstOrCreate(&ambitoInt, models.AmbitoViaje{Codigo: "INTERNACIONAL"})

	conceptoDer := models.ConceptoViaje{Codigo: "DERECHO", Nombre: "Pasaje por Derecho"}
	conceptoOfi := models.ConceptoViaje{Codigo: "OFICIAL", Nombre: "Misión Oficial"}
	configs.DB.FirstOrCreate(&conceptoDer, models.ConceptoViaje{Codigo: "DERECHO"})
	configs.DB.FirstOrCreate(&conceptoOfi, models.ConceptoViaje{Codigo: "OFICIAL"})

	tipoCupo := models.TipoSolicitud{
		Codigo:          "USO_CUPO",
		Nombre:          "Uso de Cupo Mensual",
		ConceptoViajeID: conceptoDer.ID,
	}
	configs.DB.FirstOrCreate(&tipoCupo, models.TipoSolicitud{Codigo: "USO_CUPO"})

	configs.DB.Model(&tipoCupo).Association("Ambitos").Append(&ambitoNac)

	tipoComision := models.TipoSolicitud{
		Codigo:          "COMISION",
		Nombre:          "Comisión Oficial",
		ConceptoViajeID: conceptoOfi.ID,
	}
	configs.DB.FirstOrCreate(&tipoComision, models.TipoSolicitud{Codigo: "COMISION"})
	configs.DB.Model(&tipoComision).Association("Ambitos").Append(&ambitoNac, &ambitoInt)
	tipoInvitacion := models.TipoSolicitud{
		Codigo:          "INVITACION",
		Nombre:          "Invitación Institucional",
		ConceptoViajeID: conceptoOfi.ID,
	}
	configs.DB.FirstOrCreate(&tipoInvitacion, models.TipoSolicitud{Codigo: "INVITACION"})
	configs.DB.Model(&tipoInvitacion).Association("Ambitos").Append(&ambitoNac, &ambitoInt)

	itinIdaVuelta := models.TipoItinerario{Codigo: "IDA_VUELTA", Nombre: "Ida y Vuelta"}
	itinSoloIda := models.TipoItinerario{Codigo: "SOLO_IDA", Nombre: "Solo Ida"}
	configs.DB.FirstOrCreate(&itinIdaVuelta, models.TipoItinerario{Codigo: "IDA_VUELTA"})
	configs.DB.FirstOrCreate(&itinSoloIda, models.TipoItinerario{Codigo: "SOLO_IDA"})

	fmt.Println("Catálogos sincronizados.")
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

func seedCiudades() {
	fmt.Println("Sincronizando Ciudades...")
	defaults := []models.Ciudad{
		{Nombre: "La Paz - El Alto", Code: "LPB"},
		{Nombre: "Santa Cruz - Viru Viu", Code: "VVI"},
		{Nombre: "Cochabamba - J. Wilstermann", Code: "CBB"},
		{Nombre: "Sucre - Alcantarí", Code: "SRE"},
		{Nombre: "Tarija - Cap. Oriel Lea Plaza", Code: "TJA"},
		{Nombre: "Trinidad - Tte. Jorge Henrich", Code: "TDD"},
		{Nombre: "Cobija - Cap. Aníbal Arab", Code: "CIJ"},
		{Nombre: "Oruro - Juan Mendoza", Code: "ORU"},
		{Nombre: "Potosí - Cap. Nicolás Rojas", Code: "POI"},
		{Nombre: "Uyuni - Joya Andina", Code: "UYU"},
	}

	for _, d := range defaults {
		configs.DB.Where("code = ?", d.Code).FirstOrCreate(&d)
	}
}

func seedDepartamentos() {
	fmt.Println("Sincronizando Departamentos...")
	departamentos := []models.Departamento{
		{Nombre: "LA PAZ", Codigo: "LP"},
		{Nombre: "SANTA CRUZ", Codigo: "SC"},
		{Nombre: "COCHABAMBA", Codigo: "CB"},
		{Nombre: "CHUQUISACA", Codigo: "CH"},
		{Nombre: "TARIJA", Codigo: "TJ"},
		{Nombre: "BENI", Codigo: "BE"},
		{Nombre: "PANDO", Codigo: "PA"},
		{Nombre: "ORURO", Codigo: "OR"},
		{Nombre: "POTOSI", Codigo: "PT"},
	}

	for _, d := range departamentos {
		configs.DB.Where("codigo = ?", d.Codigo).FirstOrCreate(&d)
	}
}
