package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/utils"
)

func main() {
	configs.ConnectDB()

	seedDepartamentos()
	seedProveedores()
	seedRolesAndPermissions()
	seedCatalogosViaje()
	seedDestinos()
	seedEstadosSolicitud()
	seedEstadosVoucher()
	seedEstadosPasaje()
	seedViaticosAndConfig()
	seedGeneros()
}

func assignPermsToRole(rol *models.Rol, perms []*models.Permiso) {
	if rol == nil {
		return
	}
	configs.DB.Model(rol).Association("Permisos").Replace(perms)
}

type BoaData struct {
	Data []BoaDestino `json:"Data"`
}

type BoaDestino struct {
	AirportId    int
	CodigoIata   string
	Name         string
	CountryName  string
	CityName     string
	TypeAirport  string
	IsOperated   bool
	Business     bool
	Status       string
	TimeZone     any
	DiffHourMins int
}

func seedDestinos() {
	fmt.Println("Sincronizando Destinos Defaults...")

	lpCode := "LP"
	scCode := "SC"
	cbCode := "CB"
	chCode := "CH"
	tjCode := "TJ"
	beCode := "BE"
	paCode := "PA"
	orCode := "OR"
	ptCode := "PT"

	defaults := []models.Destino{
		{IATA: "LPB", Ciudad: "La Paz", Aeropuerto: "El Alto", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &lpCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "VVI", Ciudad: "Santa Cruz", Aeropuerto: "Viru Viru", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &scCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "SRZ", Ciudad: "Santa Cruz", Aeropuerto: "El Trompillo", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &scCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "CBB", Ciudad: "Cochabamba", Aeropuerto: "Jorge Wilstermann", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &cbCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "SRE", Ciudad: "Sucre", Aeropuerto: "Alcantarí", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &chCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "TJA", Ciudad: "Tarija", Aeropuerto: "Cap. Oriel Lea Plaza", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &tjCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "TDD", Ciudad: "Trinidad", Aeropuerto: "Tte. Jorge Henrich", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &beCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "CIJ", Ciudad: "Cobija", Aeropuerto: "Cap. Aníbal Arab", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &paCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "ORU", Ciudad: "Oruro", Aeropuerto: "Juan Mendoza", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &orCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "POI", Ciudad: "Potosí", Aeropuerto: "Cap. Nicolás Rojas", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &ptCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "UYU", Ciudad: "Uyuni", Aeropuerto: "Joya Andina", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &ptCode, Pais: utils.Ptr("BOLIVIA")},

		{IATA: "RBQ", Ciudad: "Rurrenabaque", Aeropuerto: "Rurrenabaque", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &beCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "RIB", Ciudad: "Riberalta", Aeropuerto: "Cap. Selin Zeitun López", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &beCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "GYA", Ciudad: "Guayaramerín", Aeropuerto: "Cap. Emilio Beltrán", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &beCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "BYC", Ciudad: "Yacuiba", Aeropuerto: "Yacuiba", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &tjCode, Pais: utils.Ptr("BOLIVIA")},
		{IATA: "CCA", Ciudad: "Chimoré", Aeropuerto: "Chimoré", AmbitoCodigo: "NACIONAL", DepartamentoCodigo: &cbCode, Pais: utils.Ptr("BOLIVIA")},

		{IATA: "MIA", Ciudad: "Miami", Aeropuerto: "Miami International", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("ESTADOS UNIDOS")},
		{IATA: "WAS", Ciudad: "Washington D.C.", Aeropuerto: "Washington Dulles/Reagan", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("ESTADOS UNIDOS")},
		{IATA: "EZE", Ciudad: "Buenos Aires", Aeropuerto: "Ezeiza", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("ARGENTINA")},
		{IATA: "AEP", Ciudad: "Buenos Aires", Aeropuerto: "Aeroparque", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("ARGENTINA")},
		{IATA: "MAD", Ciudad: "Madrid", Aeropuerto: "Barajas", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("ESPAÑA")},
		{IATA: "GRU", Ciudad: "Sao Paulo", Aeropuerto: "Guarulhos", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("BRASIL")},
		{IATA: "BOG", Ciudad: "Bogotá", Aeropuerto: "El Dorado", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("COLOMBIA")},
		{IATA: "LIM", Ciudad: "Lima", Aeropuerto: "Jorge Chávez", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("PERU")},
		{IATA: "PTY", Ciudad: "Panamá", Aeropuerto: "Tocumen", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("PANAMA")},
		{IATA: "MEX", Ciudad: "Ciudad de México", Aeropuerto: "Benito Juárez", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("MEXICO")},
		{IATA: "SCL", Ciudad: "Santiago", Aeropuerto: "Arturo Merino Benítez", AmbitoCodigo: "INTERNACIONAL", Pais: utils.Ptr("CHILE")},
	}

	for _, d := range defaults {
		var existing models.Destino
		if err := configs.DB.Where("iata = ?", d.IATA).First(&existing).Error; err != nil {
			configs.DB.Create(&d)
		} else {
			configs.DB.Model(&existing).Updates(d)
		}
	}

	fileContent, err := os.ReadFile("cmd/seeder/data/destinos_boa.json")
	if err == nil {
		fmt.Println("Cargando destinos adicionales desde JSON...")
		var boaData BoaData
		if errLen := json.Unmarshal(fileContent, &boaData); errLen == nil {
			for _, item := range boaData.Data {
				var existing models.Destino
				if err := configs.DB.Where("iata = ?", item.CodigoIata).First(&existing).Error; err == nil {
					continue
				}

				ambito := "INTERNACIONAL"
				if item.TypeAirport == "N" {
					ambito = "NACIONAL"
				}

				newDest := models.Destino{
					IATA:         item.CodigoIata,
					Ciudad:       item.CityName,
					Aeropuerto:   item.Name,
					Pais:         &item.CountryName,
					AmbitoCodigo: ambito,
					Estado:       true,
				}
				configs.DB.Create(&newDest)
			}
		} else {
			fmt.Printf("Error parseando JSON de destinos: %v\n", errLen)
		}
	} else {
		fmt.Println("No se encontró cmd/seeder/data/destinos_boa.json, saltando carga masiva.")
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

func seedEstadosSolicitud() {
	fmt.Println("Sincronizando Estados de Solicitud...")
	estados := []models.EstadoSolicitud{
		{Codigo: "SOLICITADO", Nombre: "Solicitado", Color: "blue", Descripcion: "Solicitud creada, pendiente de aprobación"},
		{Codigo: "APROBADO", Nombre: "Aprobado", Color: "green", Descripcion: "Solicitud aprobada, pasajes en emisión"},
		{Codigo: "EMITIDO", Nombre: "Pasaje Emitido", Color: "teal", Descripcion: "Pasajes emitidos y enviados al beneficiario"},
		{Codigo: "RECHAZADO", Nombre: "Rechazado", Color: "red", Descripcion: "Solicitud rechazada por autoridad"},
		{Codigo: "FINALIZADO", Nombre: "Finalizado", Color: "gray", Descripcion: "Viaje completado y cerrado"},
	}

	for _, e := range estados {
		configs.DB.Where("codigo = ?", e.Codigo).FirstOrCreate(&e)
	}
}

func seedViaticosAndConfig() {
	fmt.Println("Sincronizando Categorías de Viáticos y Configuración...")

	categorias := []models.CategoriaViatico{
		{Nombre: "CATEGORIA 1", Codigo: 1, Monto: 360.00, Moneda: "USD", Ubicacion: "NORTE AMERICA, EUROPA, ASIA, AFRICA U OCEANIA"},
		{Nombre: "CATEGORIA 2", Codigo: 2, Monto: 300.00, Moneda: "USD", Ubicacion: "NORTE AMERICA, EUROPA, ASIA, AFRICA U OCEANIA"},
		{Nombre: "CATEGORIA 3", Codigo: 3, Monto: 276.00, Moneda: "USD", Ubicacion: "NORTE AMERICA, EUROPA, ASIA, AFRICA U OCEANIA"},
		{Nombre: "CATEGORIA 1", Codigo: 1, Monto: 300.00, Moneda: "USD", Ubicacion: "CENTRO AMERICA, SUD AMERICA O EL CARIBE"},
		{Nombre: "CATEGORIA 2", Codigo: 2, Monto: 240.00, Moneda: "USD", Ubicacion: "CENTRO AMERICA, SUD AMERICA O EL CARIBE"},
		{Nombre: "CATEGORIA 3", Codigo: 3, Monto: 207.00, Moneda: "USD", Ubicacion: "CENTRO AMERICA, SUD AMERICA O EL CARIBE"},
		{Nombre: "CATEGORIA 1", Codigo: 1, Monto: 553.00, Moneda: "Bs", Ubicacion: "INTERDEPARTAMENTAL"},
		{Nombre: "CATEGORIA 2", Codigo: 2, Monto: 465.00, Moneda: "Bs", Ubicacion: "INTERDEPARTAMENTAL"},
		{Nombre: "CATEGORIA 3", Codigo: 3, Monto: 371.00, Moneda: "Bs", Ubicacion: "INTERDEPARTAMENTAL"},
		{Nombre: "CATEGORIA 1", Codigo: 1, Monto: 332.00, Moneda: "Bs", Ubicacion: "INTRADEPARTAMENTAL"},
		{Nombre: "CATEGORIA 2", Codigo: 2, Monto: 277.00, Moneda: "Bs", Ubicacion: "INTRADEPARTAMENTAL"},
		{Nombre: "CATEGORIA 3", Codigo: 3, Monto: 222.00, Moneda: "Bs", Ubicacion: "INTRADEPARTAMENTAL"},
		{Nombre: "CATEGORIA 1", Codigo: 1, Monto: 583.00, Moneda: "Bs", Ubicacion: "FRANJA DE FRONTERA"},
		{Nombre: "CATEGORIA 2", Codigo: 2, Monto: 491.00, Moneda: "Bs", Ubicacion: "FRANJA DE FRONTERA"},
		{Nombre: "CATEGORIA 3", Codigo: 3, Monto: 391.00, Moneda: "Bs", Ubicacion: "FRANJA DE FRONTERA"},
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
		{Nombre: "SAN JOSE", Estado: true},
		{Nombre: "BOA", Estado: true},
		{Nombre: "AVEL TOURS", Estado: true},
		{Nombre: "VANGUARD TRAVEL", Estado: true},
		{Nombre: "PARANAIR", Estado: true},
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
		{Codigo: "RESPONSABLE", Nombre: "Responsable de Pasajes"},
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

	rolResponsablePermisos := []*models.Permiso{
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
	assignPermsToRole(roleMap["RESPONSABLE"], rolResponsablePermisos)
	assignPermsToRole(roleMap["ADMIN"], allPermisos)

	fmt.Println("Roles y Permisos sincronizados correctamente.")
}

func seedEstadosVoucher() {
	fmt.Println("Sincronizando Estados de Voucher...")
	estados := []models.EstadoVoucher{
		{Codigo: "DISPONIBLE", Nombre: "Disponible", Color: "green", Descripcion: "Voucher habilitado para uso"},
		{Codigo: "USADO", Nombre: "Usado", Color: "gray", Descripcion: "Voucher ya utilizado en un viaje"},
		{Codigo: "VENCIDO", Nombre: "Vencido", Color: "red", Descripcion: "Voucher expirado (fuera de fecha)"},
		{Codigo: "RESERVADO", Nombre: "Reservado", Color: "yellow", Descripcion: "Voucher en proceso de asignación"},
	}

	for _, e := range estados {
		configs.DB.Where("codigo = ?", e.Codigo).FirstOrCreate(&e)
	}
}

func seedEstadosPasaje() {
	fmt.Println("Sincronizando Estados de Pasaje...")
	estados := []models.EstadoPasaje{
		{Codigo: "EMITIDO", Nombre: "Emitido", Color: "green", Descripcion: "Pasaje emitido correctamente"},
		{Codigo: "REPROGRAMADO", Nombre: "Reprogramado", Color: "yellow", Descripcion: "Pasaje reprogramado con costo adicional"},
		{Codigo: "DEVUELTO", Nombre: "Devuelto", Color: "red", Descripcion: "Pasaje devuelto o cancelado"},
		{Codigo: "USADO", Nombre: "Usado", Color: "blue", Descripcion: "Pasaje utilizado por el viajero"},
		{Codigo: "ANULADO", Nombre: "Anulado", Color: "gray", Descripcion: "Pasaje anulado por error u otros motivos"},
		{Codigo: "NO_SE_PRESENTO", Nombre: "No se presentó", Color: "orange", Descripcion: "El pasajero no se presentó al vuelo"},
		{Codigo: "VALIDANDO_USO", Nombre: "Uso por Validar", Color: "yellow", Descripcion: "Pase a bordo subido, pendiente de validación"},
		{Codigo: "USO_RECHAZADO", Nombre: "Uso Rechazado", Color: "red", Descripcion: "Pase a bordo rechazado, debe subirse nuevamente"},
	}

	for _, e := range estados {
		configs.DB.Where("codigo = ?", e.Codigo).FirstOrCreate(&e)
	}
}

func seedGeneros() {
	fmt.Println("Sincronizando Géneros...")
	generos := []models.Genero{
		{Codigo: "F", Nombre: "Femenino"},
		{Codigo: "M", Nombre: "Masculino"},
	}

	for _, g := range generos {
		configs.DB.Where("codigo = ?", g.Codigo).FirstOrCreate(&g)
	}
}
