package app

import (
	"sistema-pasajes/internal/controllers"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/services"

	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"
)

// Container holds all instantiated services and controllers
type Container struct {
	// Services
	CupoService           *services.CupoService
	UsuarioService        *services.UsuarioService
	ReportService         *services.ReportService
	ConfiguracionService  *services.ConfiguracionService
	SolicitudService      *services.SolicitudService
	CompensacionService   *services.CompensacionService
	DescargoService       *services.DescargoService
	DescargoDerechoService *services.DescargoDerechoService
	DescargoOficialService *services.DescargoOficialService
	RolService            *services.RolService
	DestinoService        *services.DestinoService
	OrganigramaService    *services.OrganigramaService
	PeopleService         *services.PeopleService
	TipoSolicitudService  *services.TipoSolicitudService
	AmbitoService         *services.AmbitoService
	AerolineaService      *services.AerolineaService
	AgenciaService        *services.AgenciaService
	TipoItinerarioService *services.TipoItinerarioService
	RutaService           *services.RutaService
	AuthService           *services.AuthService
	PasajeService         *services.PasajeService
	ViaticoService        *services.ViaticoService
	CatViaticoService     *services.CategoriaViaticoService
	NotificationService   *services.NotificationService
	EmailService          *services.EmailService
	AlertaService         *services.AlertaService
	ConceptoService       *services.ConceptoService
	EstadoPasajeService   *services.EstadoPasajeService
	AuditService          *services.AuditService
	PushService           *services.PushService

	// Controllers
	CupoController             *controllers.CupoController
	SolicitudDerechoController *controllers.SolicitudDerechoController
	SolicitudOficialController *controllers.SolicitudOficialController
	SolicitudController        *controllers.SolicitudController
	UsuarioController          *controllers.UsuarioController
	SenadorController          *controllers.SenadorController
	FuncionarioController      *controllers.FuncionarioController
	DashboardController        *controllers.DashboardController
	CompensacionController     *controllers.CompensacionController
	DescargoDerechoController  *controllers.DescargoDerechoController
	DescargoOficialController  *controllers.DescargoOficialController
	AuthController             *controllers.AuthController
	PasajeController           *controllers.PasajeController
	PerfilController           *controllers.PerfilController
	CatalogoController         *controllers.CatalogoController
	ViaticoController          *controllers.ViaticoController
	AerolineaController        *controllers.AerolineaController
	AgenciaController          *controllers.AgenciaController
	RutaController             *controllers.RutaController
	ConfiguracionController    *controllers.ConfiguracionController
	CatCompensacionController  *controllers.CategoriaCompensacionController
	OrganigramaController      *controllers.OrganigramaController
	CategoriaViaticoController *controllers.CategoriaViaticoController
	NotificationController     *controllers.NotificationController
	LandingController          *controllers.LandingController
	AuditController            *controllers.AuditController
	ReportController           *controllers.ReportController
}

// NewContainer initializes the graph of dependencies
func NewContainer(db *gorm.DB, mongoRRHH *mongo.Database, mongoChat *mongo.Database) *Container {
	// 1. Initialize Repositories (Data Layer) - All injected with DB
	solicitudRepo := repositories.NewSolicitudRepository(db)
	aerolineaRepo := repositories.NewAerolineaRepository(db)
	cupoRepo := repositories.NewCupoDerechoRepository(db)
	userRepo := repositories.NewUsuarioRepository(db)
	itemRepo := repositories.NewCupoDerechoItemRepository(db)
	peopleRepo := repositories.NewPeopleViewRepository(mongoRRHH)
	deptoRepo := repositories.NewDepartamentoRepository(db)
	mongoUserRepo := repositories.NewMongoUserRepository(mongoChat)
	cargoRepo := repositories.NewCargoRepository(db)
	oficinaRepo := repositories.NewOficinaRepository(db)
	tipoSolicitudRepo := repositories.NewTipoSolicitudRepository(db)
	tipoItinRepo := repositories.NewTipoItinerarioRepository(db)
	codigoSecuenciaRepo := repositories.NewCodigoSecuenciaRepository(db)
	solicitudItemRepo := repositories.NewSolicitudItemRepository(db)
	pasajeRepo := repositories.NewPasajeRepository(db)
	compensacionRepo := repositories.NewCompensacionRepository(db)
	catCompensacionRepo := repositories.NewCategoriaCompensacionRepository(db)
	descargoRepo := repositories.NewDescargoRepository(db)
	rolRepo := repositories.NewRolRepository(db)
	generoRepo := repositories.NewGeneroRepository(db)
	viaticoRepo := repositories.NewViaticoRepository(db)
	catViaticoRepo := repositories.NewCategoriaViaticoRepository(db)
	zonaRepo := repositories.NewZonaViaticoRepository(db)
	configRepo := repositories.NewConfiguracionRepository(db)
	ambitoRepo := repositories.NewAmbitoViajeRepository(db)
	destinoRepo := repositories.NewDestinoRepository(db)
	agenciaRepo := repositories.NewAgenciaRepository(db)
	conceptoRepo := repositories.NewConceptoViajeRepository(db)
	rutaRepo := repositories.NewRutaRepository(db)
	notifRepo := repositories.NewNotificationRepository(db)
	estadoPasajeRepo := repositories.NewEstadoPasajeRepository(db)
	auditRepo := repositories.NewAuditRepository(db)
	pushRepo := repositories.NewPushRepository(db)

	// 2. Initialize Services (Business Layer) - Injecting Repos and simple services
	emailService := services.NewEmailService()
	auditService := services.NewAuditService(auditRepo)
	pushService := services.NewPushService(pushRepo)
	notifService := services.NewNotificationService(notifRepo, userRepo, pushService)
	configService := services.NewConfiguracionService(configRepo)
	peopleService := services.NewPeopleService(peopleRepo)
	estadoPasajeService := services.NewEstadoPasajeService(estadoPasajeRepo)

	reportService := services.NewReportService(solicitudRepo, aerolineaRepo, pasajeRepo, agenciaRepo, cupoRepo, configService)
	cupoService := services.NewCupoService(cupoRepo, userRepo, itemRepo, solicitudRepo)
	userService := services.NewUsuarioService(userRepo, peopleRepo, deptoRepo, mongoUserRepo, rolRepo, destinoRepo, cargoRepo, oficinaRepo)

	solicitudService := services.NewSolicitudService(
		solicitudRepo,
		tipoSolicitudRepo,
		userRepo,
		itemRepo,
		tipoItinRepo,
		codigoSecuenciaRepo,
		solicitudItemRepo,
		pasajeRepo,
		emailService,
		notifService,
		auditService,
	)

	compensacionService := services.NewCompensacionService(compensacionRepo, catCompensacionRepo)
	descargoService := services.NewDescargoService(descargoRepo, solicitudService, userService, auditService)
	descargoDerechoService := services.NewDescargoDerechoService(descargoRepo, solicitudService, auditService)
	descargoOficialService := services.NewDescargoOficialService(descargoRepo, solicitudService, auditService)
	rolService := services.NewRolService(rolRepo)
	destinoService := services.NewDestinoService(destinoRepo)
	organigramaService := services.NewOrganigramaService(cargoRepo, oficinaRepo)
	tipoSolicitudService := services.NewTipoSolicitudService(tipoSolicitudRepo)
	ambitoService := services.NewAmbitoService(ambitoRepo)
	aerolineaService := services.NewAerolineaService(aerolineaRepo)
	agenciaService := services.NewAgenciaService(agenciaRepo)
	tipoItinerarioService := services.NewTipoItinerarioService(tipoItinRepo)
	rutaService := services.NewRutaService(rutaRepo, destinoRepo)
	conceptoService := services.NewConceptoService(conceptoRepo)

	authService := services.NewAuthService(
		userRepo,
		mongoUserRepo,
		peopleRepo,
		rolRepo,
		generoRepo,
	)

	pasajeService := services.NewPasajeService(
		pasajeRepo,
		solicitudRepo,
		solicitudItemRepo,
		rutaRepo,
		emailService,
		auditService,
	)

	viaticoService := services.NewViaticoService(
		viaticoRepo,
		solicitudRepo,
		catViaticoRepo,
		zonaRepo,
		configService,
	)

	catViaticoService := services.NewCategoriaViaticoService(catViaticoRepo)
	alertaService := services.NewAlertaService(solicitudRepo, descargoRepo, emailService)

	// 3. Initialize Controllers - Injecting Services
	cupoCtrl := controllers.NewCupoController(cupoService, userService, reportService)

	solicitudDerechoCtrl := controllers.NewSolicitudDerechoController(
		solicitudService,
		destinoService,
		conceptoService,
		tipoSolicitudService,
		ambitoService,
		cupoService,
		userService,
		peopleService,
		reportService,
		aerolineaService,
		agenciaService,
		tipoItinerarioService,
		rutaService,
		descargoService,
	)

	solicitudOficialCtrl := controllers.NewSolicitudOficialController(
		solicitudService,
		destinoService,
		tipoSolicitudService,
		ambitoService,
		userService,
		tipoItinerarioService,
		aerolineaService,
		reportService,
		peopleService,
		descargoService,
	)

	solicitudCtrl := controllers.NewSolicitudController(solicitudService, userService)
	usuarioCtrl := controllers.NewUsuarioController(userService, auditService)
	senadorCtrl := controllers.NewSenadorController(userService, auditService)
	funcionarioCtrl := controllers.NewFuncionarioController(userService, auditService)
	dashboardCtrl := controllers.NewDashboardController(solicitudService, descargoService, userService)
	compensacionCtrl := controllers.NewCompensacionController(compensacionService, userService)

	descargoDerechoCtrl := controllers.NewDescargoDerechoController(
		descargoService,
		descargoDerechoService,
		solicitudService,
		destinoService,
		reportService,
		peopleService,
		aerolineaService,
		userService,
	)

	descargoOficialCtrl := controllers.NewDescargoOficialController(
		descargoService,
		descargoOficialService,
		solicitudService,
		destinoService,
		reportService,
		peopleService,
		configService,
	)

	authCtrl := controllers.NewAuthController(authService)
	pasajeCtrl := controllers.NewPasajeController(agenciaService, rutaService, solicitudService, pasajeService, aerolineaService)
	perfilCtrl := controllers.NewPerfilController(destinoService)
	catalogoCtrl := controllers.NewCatalogoController(tipoSolicitudService, destinoService, userService)

	viaticoCtrl := controllers.NewViaticoController(viaticoService, solicitudService, catViaticoService, reportService)
	aerolineaCtrl := controllers.NewAerolineaController(aerolineaService)
	agenciaCtrl := controllers.NewAgenciaController(agenciaService)
	rutaCtrl := controllers.NewRutaController(rutaService, aerolineaService, destinoService)
	configCtrl := controllers.NewConfiguracionController(configService, emailService)
	catCompCtrl := controllers.NewCategoriaCompensacionController(compensacionService)
	orgCtrl := controllers.NewOrganigramaController(organigramaService)
	catViaticoCtrl := controllers.NewCategoriaViaticoController(catViaticoService, viaticoService)
	notifCtrl := controllers.NewNotificationController(notifService, pushService)
	landingCtrl := controllers.NewLandingController()
	auditCtrl := controllers.NewAuditController(auditService)
	reportCtrl := controllers.NewReportController(reportService, aerolineaService, agenciaService)

	return &Container{
		// Services
		CupoService:           cupoService,
		UsuarioService:        userService,
		ReportService:         reportService,
		ConfiguracionService:  configService,
		SolicitudService:      solicitudService,
		CompensacionService:   compensacionService,
		DescargoService:       descargoService,
		DescargoDerechoService: descargoDerechoService,
		DescargoOficialService: descargoOficialService,
		RolService:            rolService,
		DestinoService:        destinoService,
		OrganigramaService:    organigramaService,
		PeopleService:         peopleService,
		TipoSolicitudService:  tipoSolicitudService,
		AmbitoService:         ambitoService,
		AerolineaService:      aerolineaService,
		AgenciaService:        agenciaService,
		TipoItinerarioService: tipoItinerarioService,
		RutaService:           rutaService,
		AuthService:           authService,
		PasajeService:         pasajeService,
		ViaticoService:        viaticoService,
		CatViaticoService:     catViaticoService,
		NotificationService:   notifService,
		EmailService:          emailService,
		AlertaService:         alertaService,
		ConceptoService:       conceptoService,
		EstadoPasajeService:   estadoPasajeService,
		AuditService:          auditService,
		PushService:           pushService,

		// Controllers
		CupoController:             cupoCtrl,
		SolicitudDerechoController: solicitudDerechoCtrl,
		SolicitudOficialController: solicitudOficialCtrl,
		SolicitudController:        solicitudCtrl,
		UsuarioController:          usuarioCtrl,
		SenadorController:          senadorCtrl,
		FuncionarioController:      funcionarioCtrl,
		DashboardController:        dashboardCtrl,
		CompensacionController:     compensacionCtrl,
		DescargoDerechoController:  descargoDerechoCtrl,
		DescargoOficialController:  descargoOficialCtrl,
		AuthController:             authCtrl,
		PasajeController:           pasajeCtrl,
		PerfilController:           perfilCtrl,
		CatalogoController:         catalogoCtrl,
		ViaticoController:          viaticoCtrl,
		AerolineaController:        aerolineaCtrl,
		AgenciaController:          agenciaCtrl,
		RutaController:             rutaCtrl,
		ConfiguracionController:    configCtrl,
		CatCompensacionController:  catCompCtrl,
		OrganigramaController:      orgCtrl,
		CategoriaViaticoController: catViaticoCtrl,
		NotificationController:     notifCtrl,
		LandingController:          landingCtrl,
		AuditController:            auditCtrl,
		ReportController:           reportCtrl,
	}
}
