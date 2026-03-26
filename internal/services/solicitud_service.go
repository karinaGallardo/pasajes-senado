package services

import (
	"context"
	"errors"
	"fmt"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func NewSolicitudService(
	repo *repositories.SolicitudRepository,
	tipoSolicitudRepo *repositories.TipoSolicitudRepository,
	usuarioRepo *repositories.UsuarioRepository,
	itemRepo *repositories.CupoDerechoItemRepository,
	tipoItinRepo *repositories.TipoItinerarioRepository,
	codigoSecuenciaRepo *repositories.CodigoSecuenciaRepository,
	solicitudItemRepo *repositories.SolicitudItemRepository,
	pasajeRepo *repositories.PasajeRepository,
	emailService *EmailService,
	notificationService *NotificationService,
	auditService *AuditService,
) *SolicitudService {
	return &SolicitudService{
		repo:                repo,
		tipoSolicitudRepo:   tipoSolicitudRepo,
		usuarioRepo:         usuarioRepo,
		itemRepo:            itemRepo,
		tipoItinRepo:        tipoItinRepo,
		codigoSecuenciaRepo: codigoSecuenciaRepo,
		solicitudItemRepo:   solicitudItemRepo,
		pasajeRepo:          pasajeRepo,
		emailService:        emailService,
		notificationService: notificationService,
		auditService:        auditService,
	}
}

type SolicitudService struct {
	repo                *repositories.SolicitudRepository
	tipoSolicitudRepo   *repositories.TipoSolicitudRepository
	usuarioRepo         *repositories.UsuarioRepository
	itemRepo            *repositories.CupoDerechoItemRepository
	tipoItinRepo        *repositories.TipoItinerarioRepository
	codigoSecuenciaRepo *repositories.CodigoSecuenciaRepository
	solicitudItemRepo   *repositories.SolicitudItemRepository
	pasajeRepo          *repositories.PasajeRepository
	emailService        *EmailService
	notificationService *NotificationService
	auditService        *AuditService
}

func (s *SolicitudService) CreateDerecho(ctx context.Context, req dtos.CreateSolicitudRequest, currentUser *models.Usuario) (*models.Solicitud, error) {
	fechaIda, err := utils.ParseDateTime(req.FechaIda)
	if err != nil {
		return nil, fmt.Errorf("error en fecha de ida: %w", err)
	}

	var fechaVuelta *time.Time
	if req.TipoItinerarioCodigo != "SOLO_IDA" {
		t, err := utils.ParseDateTime(req.FechaVuelta)
		if err != nil {
			return nil, fmt.Errorf("error en fecha de vuelta: %w", err)
		}
		fechaVuelta = t
	}

	realSolicitanteID := req.TargetUserID
	if realSolicitanteID == "" {
		realSolicitanteID = currentUser.ID
	}

	// Resolve TipoItinerario
	var tipoItin *models.TipoItinerario
	if req.TipoItinerarioCodigo != "" {
		tipoItin, _ = s.tipoItinRepo.FindByCodigo(ctx, req.TipoItinerarioCodigo)
	}
	if tipoItin == nil && req.TipoItinerario != "" {
		tipoItin, _ = s.tipoItinRepo.FindByCodigo(ctx, req.TipoItinerario)
	}
	if tipoItin == nil {
		tipoItin, _ = s.tipoItinRepo.FindByCodigo(ctx, "IDA_VUELTA")
	}

	if tipoItin == nil {
		return nil, errors.New("tipo de itinerario no válido")
	}

	solicitud := &models.Solicitud{
		BaseModel:            models.BaseModel{CreatedBy: &currentUser.ID},
		UsuarioID:            realSolicitanteID,
		TipoSolicitudCodigo:  req.TipoSolicitudCodigo,
		AmbitoViajeCodigo:    req.AmbitoViajeCodigo,
		TipoItinerarioCodigo: tipoItin.Codigo,
		Motivo:               req.Motivo,
		Autorizacion:         req.Autorizacion,
		AerolineaSugerida:    req.AerolineaSugerida,
	}

	if req.CupoDerechoItemID != "" {
		solicitud.CupoDerechoItemID = &req.CupoDerechoItemID
	}

	tipoSolicitud, err := s.tipoSolicitudRepo.FindByID(ctx, solicitud.TipoSolicitudCodigo)
	if err != nil {
		return nil, errors.New("tipo de solicitud inválido o no encontrado")
	}

	if tipoSolicitud.ConceptoViaje != nil && tipoSolicitud.ConceptoViaje.Codigo == "DERECHO" {
		beneficiary, err := s.usuarioRepo.FindByID(ctx, solicitud.UsuarioID)
		if err != nil {
			return nil, errors.New("usuario beneficiario no encontrado")
		}

		if !strings.Contains(beneficiary.Tipo, "SENADOR") {
			return nil, errors.New("solo los Senadores pueden recibir pasajes por derecho")
		}

		canCreate := false
		if currentUser.ID == beneficiary.ID ||
			currentUser.IsAdminOrResponsable() ||
			(beneficiary.EncargadoID != nil && *beneficiary.EncargadoID == currentUser.ID) {
			canCreate = true
		}

		if !canCreate {
			return nil, errors.New("no tiene autorización para emitir solicitudes de pasajes para este Senador")
		}
	}

	solicitud.EstadoSolicitudCodigo = utils.Ptr("SOLICITADO")

	// Build Items
	var items []models.SolicitudItem

	// Sempre creamos los dos tramos (IDA y VUELTA)
	// Tramo 1: IDA
	stIda := "SOLICITADO"
	if req.IdaPorConfirmar {
		stIda = "PENDIENTE"
		fechaIda = nil // Forzamos nil si es por confirmar
	}
	ida := models.SolicitudItem{
		Tipo:         models.TipoSolicitudItemIda,
		OrigenIATA:   req.OrigenIATA,
		DestinoIATA:  req.DestinoIATA,
		Fecha:        fechaIda,
		EstadoCodigo: utils.Ptr(stIda),
	}
	items = append(items, ida)

	// Tramo 2: VUELTA
	stVuelta := "SOLICITADO"
	if req.VueltaPorConfirmar {
		stVuelta = "PENDIENTE"
		fechaVuelta = nil
	}
	vuelta := models.SolicitudItem{
		Tipo:         models.TipoSolicitudItemVuelta,
		OrigenIATA:   req.DestinoIATA, // Intercambiado
		DestinoIATA:  req.OrigenIATA,  // Intercambiado
		Fecha:        fechaVuelta,
		EstadoCodigo: utils.Ptr(stVuelta),
	}
	items = append(items, vuelta)

	solicitud.Items = items
	solicitud.TipoItinerarioCodigo = "IDA_VUELTA" // Forzamos IDA_VUELTA como base

	err = s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		currentYear := time.Now().Year()
		for {
			nextVal, err := s.codigoSecuenciaRepo.GetNext(ctx, currentYear, "SPD")
			if err != nil {
				return errors.New("error generando codigo de secuencia de solicitud")
			}
			solicitud.Codigo = fmt.Sprintf("SPD-%d%04d", currentYear%100, nextVal)

			// Verificar si ya existe (incluyendo eliminados) para evitar errores de clave duplicada
			exists, _ := repoTx.ExistsByCodigo(ctx, solicitud.Codigo)
			if !exists {
				break
			}
		}

		if err := repoTx.Create(ctx, solicitud); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				// Validación de periodo vencido: solo ADMIN o RESPONSABLE pueden registrar en periodos pasados
				if item.IsVencido() && !currentUser.IsAdminOrResponsable() {
					return errors.New("el periodo de este cupo ha vencido. Solo personal administrativo puede registrar solicitudes en periodos anteriores")
				}

				item.EstadoCupoDerechoCodigo = "RESERVADO"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Fetch beneficiary to use abbreviated name in notification
	beneficiary, _ := s.usuarioRepo.FindByID(ctx, solicitud.UsuarioID)
	benefName := solicitud.UsuarioID
	if beneficiary != nil {
		benefName = beneficiary.GetNombreResumido()
	}

	s.notificationService.NotifyAdmins(ctx,
		"Nueva Solicitud: "+solicitud.Codigo,
		fmt.Sprintf("<ul class='list-none space-y-0.5 mt-1'><li><strong>Beneficiario:</strong> %s</li><li><strong>Fecha:</strong> %s</li><li><strong>Tipo:</strong> DERECHO</li></ul>",
			benefName,
			solicitud.CreatedAt.Format("02/01/2006 15:04")),
		"new_solicitud",
		fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitud.ID),
	)

	go s.sendCreationEmail(solicitud)

	return solicitud, nil
}

func (s *SolicitudService) CreateOficial(ctx context.Context, req dtos.CreateSolicitudOficialRequest, currentUser *models.Usuario) (*models.Solicitud, error) {
	realSolicitanteID := req.TargetUserID
	if realSolicitanteID == "" {
		realSolicitanteID = currentUser.ID
	}

	tipoItin, err := s.tipoItinRepo.FindByCodigo(ctx, "IDA_VUELTA")
	if err != nil || tipoItin == nil {
		return nil, errors.New("tipo de itinerario no válido")
	}

	tipoSolicitudCode := req.TipoSolicitudCodigo
	if tipoSolicitudCode == "" {
		tipoSolicitudCode = "COMISION"
	}

	solicitud := &models.Solicitud{
		BaseModel:             models.BaseModel{CreatedBy: &currentUser.ID},
		UsuarioID:             realSolicitanteID,
		TipoSolicitudCodigo:   tipoSolicitudCode,
		AmbitoViajeCodigo:     req.AmbitoViajeCodigo,
		TipoItinerarioCodigo:  tipoItin.Codigo,
		Motivo:                req.Motivo,
		Autorizacion:          req.Autorizacion,
		AerolineaSugerida:     req.AerolineaSugerida,
		EstadoSolicitudCodigo: utils.Ptr("SOLICITADO"),
	}

	// Build Items
	var items []models.SolicitudItem

	// Multi-tramo process
	for i, t := range req.Tramos {
		orig := strings.TrimSpace(t.OrigenIATA)
		dest := strings.TrimSpace(t.DestinoIATA)
		if orig == "" || dest == "" {
			continue
		}

		fSalida, err := utils.ParseDateTime(t.FechaSalida)
		if err != nil {
			return nil, fmt.Errorf("error en fecha del tramo #%d: %w", i+1, err)
		}

		tipoItem := models.TipoSolicitudItemIda
		if t.Tipo == "VUELTA" {
			tipoItem = models.TipoSolicitudItemVuelta
		}

		st := "SOLICITADO"
		if tipoItem == models.TipoSolicitudItemVuelta && fSalida == nil {
			st = "PENDIENTE"
		}

		item := models.SolicitudItem{
			Tipo:         tipoItem,
			OrigenIATA:   orig,
			DestinoIATA:  dest,
			Fecha:        fSalida,
			EstadoCodigo: utils.Ptr(st),
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		return nil, errors.New("debe agregar al menos un tramo de viaje válido")
	}

	solicitud.Items = items

	err = s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		currentYear := time.Now().Year()
		for {
			nextVal, err := s.codigoSecuenciaRepo.GetNext(ctx, currentYear, "SOF")
			if err != nil {
				return errors.New("error generando codigo de secuencia de solicitud oficial")
			}
			solicitud.Codigo = fmt.Sprintf("SOF-%d%04d", currentYear%100, nextVal)

			// Verificar duplicados (incluyendo soft-deleted)
			exists, _ := repoTx.ExistsByCodigo(ctx, solicitud.Codigo)
			if !exists {
				break
			}
		}

		if err := repoTx.Create(ctx, solicitud); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Fetch beneficiary name for notification
	beneficiary, _ := s.usuarioRepo.FindByID(ctx, realSolicitanteID)
	benefName := realSolicitanteID
	if beneficiary != nil {
		benefName = beneficiary.GetNombreResumido()
	}

	s.notificationService.NotifyAdmins(ctx,
		"Nueva Solicitud: "+solicitud.Codigo,
		fmt.Sprintf("<ul class='list-none space-y-0.5 mt-1'><li><strong>Beneficiario:</strong> %s</li><li><strong>Fecha:</strong> %s</li><li><strong>Tipo:</strong> OFICIAL</li></ul>",
			benefName,
			solicitud.CreatedAt.Format("02/01/2006 15:04")),
		"new_solicitud",
		fmt.Sprintf("/solicitudes/oficial/%s/detalle", solicitud.ID),
	)

	go s.sendCreationEmail(solicitud)

	return solicitud, nil
}

// Helper locally if not in utils

func (s *SolicitudService) sendCreationEmail(solicitud *models.Solicitud) {
	fullSol, err := s.GetByID(context.Background(), solicitud.ID)
	if err != nil {
		fmt.Printf("Error reloading solicitud for email: %v\n", err)
		return
	}

	beneficiary := fullSol.Usuario
	if beneficiary.Email == "" {
		fmt.Printf("Beneficiary %s has no email\n", beneficiary.Username)
		return
	}

	concepto := fullSol.GetConceptoNombre()
	if concepto == "" {
		concepto = "PASAJES"
	}

	subject := fmt.Sprintf("[%s] Solicitud de Pasaje Creada - %s", strings.ToUpper(concepto), fullSol.Codigo)

	tramosHTML := ""
	for _, it := range fullSol.Items {
		origen := it.OrigenIATA
		if it.Origen != nil {
			origen = it.Origen.Ciudad
		}
		destino := it.DestinoIATA
		if it.Destino != nil {
			destino = it.Destino.Ciudad
		}
		fechaIt := "-"
		if it.Fecha != nil {
			fechaIt = utils.FormatDateShortES(*it.Fecha)
		}
		tramosHTML += fmt.Sprintf(`
			<tr>
				<td style="padding: 8px; border-bottom: 1px solid #eee; width: 30%%;"><strong>Tramo %s:</strong></td>
				<td style="padding: 8px; border-bottom: 1px solid #eee;">%s &rarr; %s (%s)</td>
			</tr>`, it.Tipo, origen, destino, fechaIt)
	}

	baseURL := viper.GetString("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8284"
	}
	solPath := "derecho"
	if fullSol.GetConceptoCodigo() == "OFICIAL" {
		solPath = "oficial"
	}
	solURL := fmt.Sprintf("%s/solicitudes/%s/%s/detalle", baseURL, solPath, fullSol.ID)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333; max-width: 600px;">
			<div style="background-color: #03738C; color: white; padding: 15px; border-radius: 5px 5px 0 0;">
				<h2 style="margin:0;">Solicitud de Pasaje Registrada</h2>
				<p style="margin: 5px 0 0 0; opacity: 0.9;">Concepto: %s</p>
			</div>
			<div style="padding: 20px; border: 1px solid #ddd; border-top: none; border-radius: 0 0 5px 5px;">
				<p>Hola <strong>%s</strong>,</p>
				<p>Se ha registrado una nueva solicitud de pasaje a su nombre.</p>

				<h3 style="color: #03738C; margin-bottom: 10px;">Información General</h3>
				<table style="border-collapse: collapse; width: 100%%; margin-bottom: 20px;">
					<tr>
						<td style="padding: 8px; border-bottom: 1px solid #eee; width: 30%%;"><strong>Código:</strong></td>
						<td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Concepto:</strong></td>
						<td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Estado:</strong></td>
						<td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td>
					</tr>
				</table>

				<h3 style="color: #03738C; margin-bottom: 10px;">Itinerario Solicitado</h3>
				<table style="border-collapse: collapse; width: 100%%; margin-bottom: 20px;">
					%s
				</table>

				<div style="margin-top: 25px; text-align: center;">
					<a href="%s" target="_blank" style="background-color: #03738C; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold;">Ver Detalles en el Sistema</a>
				</div>
				<p style="font-size: 11px; color: #999; margin-top: 20px; text-align: center;">
					Si el botón no funciona/no se ve, copie esta URL:<br>
					%s
				</p>
			</div>
		</div>
	`, strings.ToUpper(concepto), beneficiary.GetNombreCompleto(), fullSol.Codigo, concepto, fullSol.GetEstado(), tramosHTML, solURL, solURL)

	err = s.emailService.SendEmail([]string{beneficiary.Email}, nil, nil, subject, body)
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
	}
}

func (s *SolicitudService) sendRevertApprovalEmail(solicitud *models.Solicitud) {
	fullSol, err := s.GetByID(context.Background(), solicitud.ID)
	if err != nil {
		fmt.Printf("Error reloading solicitud for email: %v\n", err)
		return
	}

	beneficiary := fullSol.Usuario
	if beneficiary.Email == "" {
		return
	}

	concepto := fullSol.GetConceptoNombre()
	if concepto == "" {
		concepto = "PASAJES"
	}

	subject := fmt.Sprintf("[%s] Solicitud de Pasaje Revertida - %s", strings.ToUpper(concepto), fullSol.Codigo)

	baseURL := viper.GetString("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8284"
	}
	solPath := "derecho"
	if fullSol.GetConceptoCodigo() == "OFICIAL" {
		solPath = "oficial"
	}
	solURL := fmt.Sprintf("%s/solicitudes/%s/%s/detalle", baseURL, solPath, fullSol.ID)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333; max-width: 600px;">
			<div style="background-color: #d97706; color: white; padding: 15px; border-radius: 5px 5px 0 0;">
				<h2 style="margin:0;">Aprobación Revertida</h2>
				<p style="margin: 5px 0 0 0; opacity: 0.9;">Concepto: %s</p>
			</div>
			<div style="padding: 20px; border: 1px solid #ddd; border-top: none; border-radius: 0 0 5px 5px;">
				<p>Hola <strong>%s</strong>,</p>
				<p>La aprobación de su solicitud <strong>%s</strong> ha sido revertida al estado <strong>SOLICITADO</strong>.</p>
				<p>Concepto de la solicitud: <strong>%s</strong></p>
				<p>Esto puede deberse a ajustes necesarios en el itinerario o cambios de último momento.</p>

				<div style="margin-top: 25px; text-align: center;">
					<a href="%s" target="_blank" style="background-color: #d97706; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold;">Ver Solicitud</a>
				</div>
				<p style="font-size: 11px; color: #999; margin-top: 20px; text-align: center;">
					Si el botón no funciona/no se ve, copie esta URL:<br>
					%s
				</p>
			</div>
		</div>
	`, strings.ToUpper(concepto), beneficiary.GetNombreCompleto(), fullSol.Codigo, concepto, solURL, solURL)

	_ = s.emailService.SendEmail([]string{beneficiary.Email}, nil, nil, subject, body)
}

func (s *SolicitudService) GetAll(ctx context.Context, status string, concepto string) ([]models.Solicitud, error) {
	return s.repo.FindAll(ctx, status, concepto)
}

func (s *SolicitudService) GetByUserID(ctx context.Context, userID string, status string, concepto string) ([]models.Solicitud, error) {
	return s.repo.FindByUserID(ctx, userID, status, concepto)
}

func (s *SolicitudService) GetByUserIdOrAccesibleByEncargadoID(ctx context.Context, userID string, status string, concepto string) ([]models.Solicitud, error) {
	return s.repo.FindByUserIdOrAccesibleByEncargadoID(ctx, userID, status, concepto)
}

func (s *SolicitudService) GetPaginated(ctx context.Context, userID string, isAdmin bool, status string, concepto string, page, limit int, searchTerm string) (*repositories.PaginatedSolicitudes, error) {
	return s.repo.FindPaginated(ctx, userID, isAdmin, status, concepto, page, limit, searchTerm)
}

func (s *SolicitudService) GetPendientesDescargo(ctx context.Context, userID string, isAdmin bool) ([]models.Solicitud, error) {
	return s.repo.FindPendientesDeDescargoUI(ctx, userID, isAdmin)
}

func (s *SolicitudService) GetPendientesDescargoPaginated(ctx context.Context, userID string, isAdmin bool, page, limit int, searchTerm string) (*repositories.PaginatedSolicitudes, error) {
	return s.repo.FindPendientesDeDescargoPaginated(ctx, userID, isAdmin, page, limit, searchTerm)
}

func (s *SolicitudService) GetByCupoDerechoItemID(ctx context.Context, itemID string) ([]models.Solicitud, error) {
	return s.repo.FindByCupoDerechoItemID(ctx, itemID)
}

func (s *SolicitudService) GetByID(ctx context.Context, id string) (*models.Solicitud, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *SolicitudService) GetItemByID(ctx context.Context, id string) (*models.SolicitudItem, error) {
	return s.solicitudItemRepo.FindByID(ctx, id)
}

func (s *SolicitudService) Approve(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		hasIda := false
		hasVuelta := false
		for i := range solicitud.Items {
			switch solicitud.Items[i].Tipo {
			case models.TipoSolicitudItemIda:
				hasIda = true
			case models.TipoSolicitudItemVuelta:
				hasVuelta = true
			}
		}

		approveAll := hasIda && hasVuelta

		for i := range solicitud.Items {
			item := &solicitud.Items[i]
			shouldApprove := false

			if approveAll {
				shouldApprove = true
			} else {
				// Approve only if it has a confirmed date (i.e., not PENDING placeholder)
				// Re-using the PENDIENTE status logic we added
				if item.GetEstado() != "PENDIENTE" {
					shouldApprove = true
				}
			}

			if shouldApprove && item.GetEstado() == "SOLICITADO" {
				st := "APROBADO"
				item.EstadoCodigo = &st
				if err := tx.Save(item).Error; err != nil {
					return err
				}
			}
		}

		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "RESERVADO"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}

        s.auditService.Log(ctx, "APROBAR_SOLICITUD", "solicitud", solicitud.ID, "SOLICITADO", "APROBADO", "", "")
		return nil
	})
}

func (s *SolicitudService) RevertApproval(ctx context.Context, id string) error {
	var solicitudForEmail *models.Solicitud
	err := s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		st := ""
		if solicitud.EstadoSolicitudCodigo != nil {
			st = *solicitud.EstadoSolicitudCodigo
		}

		if st != "APROBADO" && st != "PARCIALMENTE_APROBADO" && st != "EMITIDO" {
			return errors.New("la solicitud no está en un estado que permita revertir la aprobación (" + st + ")")
		}

		hasPasajes := false
		for _, t := range solicitud.Items {
			if len(t.Pasajes) > 0 {
				hasPasajes = true
				break
			}
		}

		if hasPasajes {
			return errors.New("no se puede revertir la aprobación porque ya tiene pasajes asignados")
		}

		// Revert items
		for i := range solicitud.Items {
			item := &solicitud.Items[i]
			if item.GetEstado() == "APROBADO" {
				st := "SOLICITADO"
				item.EstadoCodigo = &st
				if err := tx.Save(item).Error; err != nil {
					return err
				}
			}
		}

		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}
		solicitudForEmail = solicitud
		s.auditService.Log(ctx, "REVERTIR_APROBACION", "solicitud", solicitud.ID, st, "SOLICITADO", "", "")
		return nil
	})

	if err == nil && solicitudForEmail != nil {
		go s.sendRevertApprovalEmail(solicitudForEmail)
	}

	return err
}

func (s *SolicitudService) Finalize(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		if err := repoTx.UpdateStatus(ctx, id, "FINALIZADO"); err != nil {
			return err
		}

		// Mark all items as FINALIZADO and their pasajes as USADO
		for _, item := range solicitud.Items {
			if err := tx.Model(&item).Update("estado_codigo", "FINALIZADO").Error; err != nil {
				return err
			}

			// Mark only EMITIDO pasajes as USADO
			for _, p := range item.Pasajes {
				if p.GetEstadoCodigo() == "EMITIDO" {
					if err := tx.Model(&p).Update("estado_pasaje_codigo", "USADO").Error; err != nil {
						return err
					}
				}
			}
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "USADO"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (s *SolicitudService) Reject(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		// Reject all items
		for i := range solicitud.Items {
			item := &solicitud.Items[i]
			st := "RECHAZADO"
			item.EstadoCodigo = &st
			if err := tx.Save(item).Error; err != nil {
				return err
			}
		}

		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "DISPONIBLE"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}
		s.auditService.Log(ctx, "RECHAZAR_SOLICITUD", "solicitud", solicitud.ID, "SOLICITADO", "RECHAZADO", "", "")
		return nil
	})
}

func (s *SolicitudService) Update(ctx context.Context, solicitud *models.Solicitud) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		// 1. Obtener el estado ACTUAL de la DB
		var dbSol models.Solicitud
		if err := tx.Preload("Items").First(&dbSol, "id = ?", solicitud.ID).Error; err != nil {
			return err
		}

		dbItems := make(map[string]models.SolicitudItem)
		for _, it := range dbSol.Items {
			dbItems[it.ID] = it
		}

		// 2. Procesar tramos
		for i := range solicitud.Items {
			item := &solicitud.Items[i]
			old, exists := dbItems[item.ID]

			if !exists {
				fmt.Printf("DEBUG: Tramo nuevo detectado ID=%s\n", item.ID)
				if err := tx.Save(item).Error; err != nil {
					return err
				}
				continue
			}

			changes := item.GetChanges(old)
			if len(changes) > 0 {
				fmt.Printf("DEBUG: Cambios en Tramo %s (%s): %v\n", item.ID, item.Tipo, changes)
				if err := tx.Model(item).Updates(changes).Error; err != nil {
					return err
				}
			}
		}

		// 3. Actualizar Solicitud (Padre)
		solicitud.UpdateStatusBasedOnItems()
		parentChanges := solicitud.GetChanges(dbSol)

		if len(parentChanges) > 0 {
			fmt.Printf("DEBUG: Cambios en Solicitud %s: %v\n", solicitud.Codigo, parentChanges)
			if err := tx.Model(solicitud).Updates(parentChanges).Error; err != nil {
				return err
			}
		} else {
			fmt.Printf("DEBUG: No se detectaron cambios en Solicitud %s\n", solicitud.Codigo)
		}

		return nil
	})
}

func (s *SolicitudService) Delete(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(ctx, id)
		if err == nil && solicitud != nil && solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "DISPONIBLE"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}

		return repoTx.Delete(ctx, id)
	})
}

func (s *SolicitudService) ApproveItem(ctx context.Context, solicitudID, itemID string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(ctx, solicitudID)
		if err != nil {
			return err
		}
		found := false
		for i := range solicitud.Items {
			if solicitud.Items[i].ID == itemID {
				st := "APROBADO"
				solicitud.Items[i].EstadoCodigo = &st
				if err := tx.Save(&solicitud.Items[i]).Error; err != nil {
					return err
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("item no encontrado")
		}
		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}

		// Si es derecho, marcar como reservado
		if solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "RESERVADO"
				itemRepoTx.Update(ctx, item)
			}
		}
		return nil
	})
}

func (s *SolicitudService) RejectItem(ctx context.Context, solicitudID, itemID string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(ctx, solicitudID)
		if err != nil {
			return err
		}
		found := false
		for i := range solicitud.Items {
			if solicitud.Items[i].ID == itemID {
				st := "RECHAZADO"
				solicitud.Items[i].EstadoCodigo = &st
				if err := tx.Save(&solicitud.Items[i]).Error; err != nil {
					return err
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("item no encontrado")
		}
		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}
		// Si todos los items activos son rechazados, liberar el cupo
		allRejected := true
		for _, it := range solicitud.Items {
			st := it.GetEstado()
			if st != "RECHAZADO" && st != "CANCELADO" && st != "PENDIENTE" {
				allRejected = false
				break
			}
		}
		if allRejected && solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "DISPONIBLE"
				itemRepoTx.Update(ctx, item)
			}
		}
		return nil
	})
}

func (s *SolicitudService) RevertApprovalItem(ctx context.Context, solicitudID, itemID string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		// 1. Verificar si existe la solicitud
		sol, err := repoTx.FindByID(ctx, solicitudID)
		if err != nil {
			return err
		}

		// 2. Verificar si el tramo (item) existe en esta solicitud
		var item *models.SolicitudItem
		for i := range sol.Items {
			if sol.Items[i].ID == itemID {
				item = &sol.Items[i]
				break
			}
		}
		if item == nil {
			return fmt.Errorf("tramo no encontrado")
		}

		// 3. Verificar si tiene pasajes activos o emitidos (distintos de ANULADO)
		// Usamos el helper HasActivePasaje que ya ignora los anulados
		if item.HasActivePasaje() {
			return fmt.Errorf("no se puede revertir: tiene pasajes activos o emitidos. Debe anularlos primero.")
		}

		// 4. Revertir aprobación
		nuevoEstado := "SOLICITADO"
		item.EstadoCodigo = &nuevoEstado
		if err := tx.Save(item).Error; err != nil {
			return err
		}

		// Sincronización de estados
		sol.UpdateStatusBasedOnItems()
		if err := repoTx.Update(ctx, sol); err != nil {
			return err
		}

		return nil
	})
}

func (s *SolicitudService) ReprogramarItem(ctx context.Context, req dtos.ReprogramarSolicitudItemRequest) error {
	stateReprog := "REPROGRAMADO"
	pasajeAnulado := "ANULADO"
	// Combine Fecha(2006-01-02) and Hora(15:04) into a single timestamp
	fechaStr := req.Fecha + " " + req.Hora
	fecha, err := utils.ParseDateTime(fechaStr)
	if err != nil {
		return fmt.Errorf("formato de fecha/hora inválido: %w", err)
	}

	return s.repo.RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		// 1. Find old Item
		var oldItem models.SolicitudItem
		if err := tx.First(&oldItem, "id = ?", req.SolicitudItemID).Error; err != nil {
			return fmt.Errorf("item de solicitud no encontrado: %w", err)
		}

		// 2. Mark old item as REPROGRAMADO
		if err := tx.Model(&oldItem).Update("estado_codigo", stateReprog).Error; err != nil {
			return fmt.Errorf("error marcando item anterior como reprogramado: %w", err)
		}

		// 3. Find and annul any associated Pasajes (issued tickets)
		var pasajes []models.Pasaje
		if err := tx.Find(&pasajes, "solicitud_item_id = ? AND estado_pasaje_codigo = 'EMITIDO'", oldItem.ID).Error; err == nil {
			for i := range pasajes {
				p := &pasajes[i]
				p.EstadoPasajeCodigo = &pasajeAnulado
				if req.Motivo != "" {
					p.Glosa += " | Reprogramado. Motivo: " + req.Motivo
				}
				if err := tx.Save(p).Error; err != nil {
					return fmt.Errorf("error anulando pasaje: %w", err)
				}
			}
		}

		// 4. Create NEW SolicitudItem
		newItem := models.SolicitudItem{
			SolicitudID:  oldItem.SolicitudID,
			Tipo:         oldItem.Tipo,
			OrigenIATA:   oldItem.OrigenIATA,
			DestinoIATA:  oldItem.DestinoIATA,
			Fecha:        fecha,
			EstadoCodigo: utils.Ptr("SOLICITADO"),
		}

		if err := tx.Create(&newItem).Error; err != nil {
			return fmt.Errorf("error creando nuevo item de solicitud: %w", err)
		}

		return nil
	})
}
func (s *SolicitudService) UpdateOficial(ctx context.Context, id string, req dtos.CreateSolicitudOficialRequest) error {
	// Minimal validation: at least one tramo
	if len(req.Tramos) == 0 {
		return errors.New("debe agregar al menos un tramo de viaje válido")
	}

	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		// 1. Load the existing solicitation
		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		// 2. Update parent fields using Updates to avoid side effects on associations
		updates := map[string]any{
			"tipo_solicitud_codigo": req.TipoSolicitudCodigo,
			"motivo":                req.Motivo,
			"autorizacion":          req.Autorizacion,
			"ambito_viaje_codigo":   req.AmbitoViajeCodigo,
			"aerolinea_sugerida":    req.AerolineaSugerida,
		}

		if err := tx.Model(solicitud).Updates(updates).Error; err != nil {
			return err
		}

		// 3. Map existing items by ID for quick lookup
		existingItemsMap := make(map[string]*models.SolicitudItem)
		for i := range solicitud.Items {
			existingItemsMap[solicitud.Items[i].ID] = &solicitud.Items[i]
		}

		var itemsToKeepIDs []string

		// 4. Process tramos from request
		for i, t := range req.Tramos {
			orig := strings.TrimSpace(t.OrigenIATA)
			dest := strings.TrimSpace(t.DestinoIATA)
			if orig == "" || dest == "" {
				continue
			}

			fSalida, err := utils.ParseDateTime(t.FechaSalida)
			if err != nil {
				return fmt.Errorf("error en fecha del tramo #%d: %w", i+1, err)
			}

			if t.ID != "" {
				// Updating an existing item
				if existing, ok := existingItemsMap[t.ID]; ok {
					itemsToKeepIDs = append(itemsToKeepIDs, t.ID)

					// Only allow updates if current status is editable
					st := existing.GetEstado()
					if st == "SOLICITADO" || st == "RECHAZADO" || st == "PENDIENTE" {
						existing.OrigenIATA = orig
						existing.DestinoIATA = dest
						existing.Fecha = fSalida

						// Clear relations to force GORM to use the new IATA codes
						existing.Origen = nil
						existing.Destino = nil

						if t.Tipo == "VUELTA" {
							existing.Tipo = models.TipoSolicitudItemVuelta
						} else {
							existing.Tipo = models.TipoSolicitudItemIda
						}

						if err := tx.Save(existing).Error; err != nil {
							return err
						}
					}
				}
			} else {
				// Creating a new item
				st := "SOLICITADO"
				tipoItem := models.TipoSolicitudItemIda
				if t.Tipo == "VUELTA" {
					tipoItem = models.TipoSolicitudItemVuelta
					if fSalida == nil {
						st = "PENDIENTE"
					}
				}

				newItem := models.SolicitudItem{
					SolicitudID:  id,
					Tipo:         tipoItem,
					OrigenIATA:   orig,
					DestinoIATA:  dest,
					Fecha:        fSalida,
					EstadoCodigo: utils.Ptr(st),
				}

				if err := tx.Create(&newItem).Error; err != nil {
					return err
				}
			}
		}

		// 5. Delete "SOLICITADO" items that were removed in the UI
		deleteQuery := tx.Where("solicitud_id = ? AND estado_codigo = ?", id, "SOLICITADO")
		if len(itemsToKeepIDs) > 0 {
			deleteQuery = deleteQuery.Where("id NOT IN ?", itemsToKeepIDs)
		}
		if err := deleteQuery.Delete(&models.SolicitudItem{}).Error; err != nil {
			return err
		}

		// 6. Refresh the solicitation to recalculate its global status
		var finalizedSol models.Solicitud
		if err := tx.Preload("Items").Preload("Items.Pasajes").First(&finalizedSol, "id = ?", id).Error; err == nil {
			finalizedSol.UpdateStatusBasedOnItems()
			if err := tx.Model(&finalizedSol).Update("estado_solicitud_codigo", finalizedSol.EstadoSolicitudCodigo).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *SolicitudService) RevertFinalize(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		if solicitud.EstadoSolicitudCodigo == nil || *solicitud.EstadoSolicitudCodigo != "FINALIZADO" {
			return errors.New("la solicitud no está en estado FINALIZADO")
		}

		// Change status back to EMITIDO (since it reached FINALIZADO, it must have been EMITIDO)
		if err := repoTx.UpdateStatus(ctx, id, "EMITIDO"); err != nil {
			return err
		}

		// Revert items to EMITIDO and their pasajes back to EMITIDO
		for _, item := range solicitud.Items {
			if err := tx.Model(&item).Update("estado_codigo", "EMITIDO").Error; err != nil {
				return err
			}

			// Revert USADO pasajes back to EMITIDO
			for _, p := range item.Pasajes {
				if p.GetEstadoCodigo() == "USADO" {
					if err := tx.Model(&p).Update("estado_pasaje_codigo", "EMITIDO").Error; err != nil {
						return err
					}
				}
			}
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "RESERVADO"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}

		return nil
	})
}
func (s *SolicitudService) GetPendingCount(ctx context.Context, userID string, isAdmin bool) (int64, error) {
	return s.repo.CountPending(ctx, userID, isAdmin)
}

func (s *SolicitudService) UpdateSolicitudDates(ctx context.Context, id string, createdAt, updatedAt string) error {
	cA, err := utils.ParseDateTime(createdAt)
	if err != nil {
		return fmt.Errorf("fecha de creación inválida: %w", err)
	}

	uA, err := utils.ParseDateTime(updatedAt)
	if err != nil {
		return fmt.Errorf("fecha de actualización inválida: %w", err)
	}

	return s.repo.UpdateTimestamps(ctx, id, cA, uA)
}

func (s *SolicitudService) UpdateSolicitudItemDates(ctx context.Context, id string, createdAt, updatedAt string) error {
	cA, err := utils.ParseDateTime(createdAt)
	if err != nil {
		return fmt.Errorf("fecha de creación inválida: %w", err)
	}

	uA, err := utils.ParseDateTime(updatedAt)
	if err != nil {
		return fmt.Errorf("fecha de actualización inválida: %w", err)
	}

	return s.solicitudItemRepo.UpdateTimestamps(ctx, id, cA, uA)
}
