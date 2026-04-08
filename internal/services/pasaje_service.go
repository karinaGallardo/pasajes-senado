package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"sistema-pasajes/internal/worker"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type PasajeService struct {
	repo              *repositories.PasajeRepository
	solicitudRepo     *repositories.SolicitudRepository
	solicitudItemRepo *repositories.SolicitudItemRepository
	rutaRepo          *repositories.RutaRepository
	emailService      *EmailService
	auditService      *AuditService
}

func NewPasajeService(
	repo *repositories.PasajeRepository,
	solicitudRepo *repositories.SolicitudRepository,
	solicitudItemRepo *repositories.SolicitudItemRepository,
	rutaRepo *repositories.RutaRepository,
	emailService *EmailService,
	auditService *AuditService,
) *PasajeService {
	return &PasajeService{
		repo:              repo,
		solicitudRepo:     solicitudRepo,
		solicitudItemRepo: solicitudItemRepo,
		rutaRepo:          rutaRepo,
		emailService:      emailService,
		auditService:      auditService,
	}
}

func (s *PasajeService) Create(ctx context.Context, solicitudID string, req dtos.CreatePasajeRequest, filePath string) (*models.Pasaje, error) {
	if filePath == "" {
		return nil, fmt.Errorf("el documento del pasaje (PDF) es obligatorio")
	}

	costo := utils.ParseFloat(req.Costo)
	fechaVueloPtr, err := utils.ParseDateTime(req.FechaVuelo)
	var fechaVuelo time.Time
	if err == nil && fechaVueloPtr != nil {
		fechaVuelo = *fechaVueloPtr
	}

	var aerolineaID *string
	if req.AerolineaID != "" {
		aerolineaID = &req.AerolineaID
	}

	status := "REGISTRADO"

	if req.NumeroBillete != "" {
		existing, _ := s.repo.FindByNumeroBillete(ctx, req.NumeroBillete)
		if existing != nil && existing.ID != "" {
			return nil, fmt.Errorf("ya existe un pasaje con el número de billete %s", req.NumeroBillete)
		}
	}

	// Rule: One active pasaje per item
	if req.SolicitudItemID != "" {
		item, err := s.solicitudItemRepo.FindByID(ctx, req.SolicitudItemID)
		if err == nil && item != nil {
			if item.HasActivePasaje() {
				return nil, fmt.Errorf("este tramo ya tiene un pasaje activo. Debe anular el anterior antes de asignar uno nuevo.")
			}
		}
	}

	pasaje := &models.Pasaje{
		SolicitudID:        solicitudID,
		SolicitudItemID:    &req.SolicitudItemID,
		EstadoPasajeCodigo: &status,
		AerolineaID:        aerolineaID,
		AgenciaID:          &req.AgenciaID,
		NumeroVuelo:        req.NumeroVuelo,
		RutaID:             utils.NilIfEmpty(req.RutaID),
		FechaVuelo:         fechaVuelo,
		CodigoReserva:      req.CodigoReserva,
		NumeroBillete:      req.NumeroBillete,
		NumeroFactura:      req.NumeroFactura,
		Glosa:              req.Glosa,
		Costo:              costo,
		CostoUtilizacion:   costo,
		Diferencia:         0.0,
		Archivo:            filePath,
	}

	if req.FechaEmision != "" {
		fe := utils.ParseDate("2006-01-02", req.FechaEmision)
		pasaje.FechaEmision = &fe
	}

	err = s.repo.RunTransaction(func(repo *repositories.PasajeRepository, tx *gorm.DB) error {
		if err := repo.Create(ctx, pasaje); err != nil {
			return err
		}

		// Paranoid check: ensure saved record has file
		if pasaje.Archivo == "" {
			return fmt.Errorf("falta el documento del pasaje, por eso no se guardó el registro")
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Email notifications are now handled upon "EMITIDO" status update, not creation.

	return pasaje, nil
}

func (s *PasajeService) GetBySolicitudID(ctx context.Context, solicitudID string) ([]models.Pasaje, error) {
	return s.repo.FindBySolicitudID(ctx, solicitudID)
}

func (s *PasajeService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}
func (s *PasajeService) GetByID(ctx context.Context, id string) (*models.Pasaje, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *PasajeService) UpdateFromRequest(ctx context.Context, req dtos.UpdatePasajeRequest, archivo string, paseAbordo string) error {
	pasaje, err := s.repo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}

	// Validate duplicate ticket number
	if req.NumeroBillete != "" {
		existing, _ := s.repo.FindByNumeroBillete(ctx, req.NumeroBillete)
		if existing != nil && existing.ID != "" && existing.ID != req.ID {
			return fmt.Errorf("ya existe un pasaje con el número de billete %s", req.NumeroBillete)
		}
	}

	pasaje.NumeroVuelo = req.NumeroVuelo
	pasaje.RutaID = utils.NilIfEmpty(req.RutaID)
	pasaje.NumeroBillete = req.NumeroBillete
	pasaje.NumeroFactura = req.NumeroFactura
	pasaje.CodigoReserva = req.CodigoReserva
	pasaje.Glosa = req.Glosa

	if req.AerolineaID != "" {
		pasaje.AerolineaID = &req.AerolineaID
	}
	if req.AgenciaID != "" {
		pasaje.AgenciaID = &req.AgenciaID
	}

	pasaje.Costo = utils.ParseFloat(req.Costo)
	pasaje.CostoUtilizacion = pasaje.Costo
	pasaje.Diferencia = 0.0

	if fvPtr, err := utils.ParseDateTime(req.FechaVuelo); err == nil && fvPtr != nil {
		pasaje.FechaVuelo = *fvPtr
	}

	if req.FechaEmision != "" {
		fe := utils.ParseDate("2006-01-02", req.FechaEmision)
		pasaje.FechaEmision = &fe
	} else {
		pasaje.FechaEmision = nil
	}

	if archivo != "" {
		pasaje.Archivo = archivo
	}
	if paseAbordo != "" {
		pasaje.ArchivoPaseAbordo = paseAbordo
	}

	// Ensure pasaje has an archive
	if pasaje.Archivo == "" {
		return fmt.Errorf("el pasaje debe tener un archivo PDF asociado")
	}

	return s.repo.Update(ctx, pasaje)
}

func (s *PasajeService) Update(ctx context.Context, pasaje *models.Pasaje) error {
	return s.repo.Update(ctx, pasaje)
}

func (s *PasajeService) DevolverPasaje(ctx context.Context, req dtos.DevolverPasajeRequest) error {
	pasaje, err := s.repo.FindByID(ctx, req.PasajeID)
	if err != nil {
		return err
	}

	costoPenalidad := utils.ParseFloat(req.CostoPenalidad)
	pasaje.EstadoPasajeCodigo = utils.Ptr("ANULADO")

	if pasaje.Glosa != "" {
		pasaje.Glosa += " | Devolución: " + req.Glosa
	} else {
		pasaje.Glosa = "Devolución: " + req.Glosa
	}
	pasaje.CostoPenalidad = costoPenalidad

	return s.repo.Update(ctx, pasaje)
}

func (s *PasajeService) UpdateStatus(ctx context.Context, id string, status string, ticketPath string, pasePath string) error {
	pasaje, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	oldStatus := ""
	if pasaje.EstadoPasajeCodigo != nil {
		oldStatus = *pasaje.EstadoPasajeCodigo
	}

	pasaje.EstadoPasajeCodigo = &status
	if ticketPath != "" {
		pasaje.Archivo = ticketPath
	}
	if pasePath != "" {
		pasaje.ArchivoPaseAbordo = pasePath
	}

	if err := s.repo.Update(ctx, pasaje); err != nil {
		return err
	}

	s.auditService.Log(ctx, "CAMBIAR_ESTADO_PASAJE", "pasaje", id, "", status, "", "")

	// If Pasaje is EMITIDO, also update Request Item state to EMITIDO
	if status == "EMITIDO" && pasaje.SolicitudItemID != nil {
		s.solicitudItemRepo.UpdateStatus(ctx, *pasaje.SolicitudItemID, "EMITIDO")

		// Recalculate Solicitud global status via GORM Hooks
		sol, err := s.solicitudRepo.FindByID(ctx, pasaje.SolicitudID)
		if err == nil && sol != nil {
			s.solicitudRepo.Update(ctx, sol)
		}
	}

	// Reversion logic: if it was EMITIDO and now REGISTRADO
	if oldStatus == "EMITIDO" && status == "REGISTRADO" && pasaje.SolicitudItemID != nil {
		// Revert item back to APROBADO (previous state)
		s.solicitudItemRepo.UpdateStatus(ctx, *pasaje.SolicitudItemID, "APROBADO")

		// Recalculate Solicitud global status via GORM Hooks
		sol, err := s.solicitudRepo.FindByID(ctx, pasaje.SolicitudID)
		if err == nil && sol != nil {
			s.solicitudRepo.Update(ctx, sol)
		}
	}

	// Trigger email if emitted
	if status == "EMITIDO" {
		worker.GetPool().Submit(&EmissionEmailJob{
			Service:  s,
			PasajeID: id,
		})
	}

	// Trigger email if reverted
	if oldStatus == "EMITIDO" && status == "REGISTRADO" {
		worker.GetPool().Submit(&ReversionEmailJob{
			Service:  s,
			PasajeID: id,
		})
	}

	return nil
}

// EmissionEmailJob encapsula la tarea de enviar un correo de emisión.
type EmissionEmailJob struct {
	Service  *PasajeService
	PasajeID string
}

func (j *EmissionEmailJob) Name() string {
	return "EmissionEmailJob:" + j.PasajeID
}

func (j *EmissionEmailJob) Run(ctx context.Context) error {
	// Recargamos el pasaje con preloads para tener toda la info (usuario, etc)
	p, err := j.Service.repo.WithContext(ctx).FindByID(ctx, j.PasajeID)
	if err != nil {
		return err
	}

	sol, err := j.Service.solicitudRepo.WithContext(ctx).FindByID(ctx, p.SolicitudID)
	if err != nil || sol == nil {
		return err
	}

	j.Service.sendEmissionEmail(sol, p)
	return nil
}

func (s *PasajeService) sendEmissionEmail(sol *models.Solicitud, pasaje *models.Pasaje) {
	usuario := sol.Usuario
	if usuario.Email == "" {
		return
	}

	to := []string{usuario.Email}
	var cc []string

	if usuario.Encargado != nil && usuario.Encargado.Email != "" {
		cc = append(cc, usuario.Encargado.Email)
	}

	concepto := sol.GetConceptoNombre()
	if concepto == "" {
		concepto = "PASAJES"
	}

	tipoTramoStr := "Tramo"
	if pasaje.SolicitudItem != nil {
		tipoTramoStr = string(pasaje.SolicitudItem.Tipo)
	}

	subject := fmt.Sprintf("[%s] Pasaje Emitido (%s) - Solicitud %s", strings.ToUpper(concepto), tipoTramoStr, sol.Codigo)

	ruta := pasaje.GetRutaDisplay()
	fecha := utils.FormatDateTimeLongES(pasaje.FechaVuelo)
	billete := pasaje.NumeroBillete
	aerolinea := "N/A"
	if pasaje.Aerolinea != nil {
		aerolinea = pasaje.Aerolinea.Nombre
	}
	vuelo := pasaje.NumeroVuelo

	baseURL := viper.GetString("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8284"
	}
	cleanPath := strings.ReplaceAll(pasaje.Archivo, "\\", "/")
	fileURL := fmt.Sprintf("%s/%s", baseURL, cleanPath)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333; max-width: 600px;">
			<div style="background-color: #03738C; color: white; padding: 15px; border-radius: 5px 5px 0 0;">
				<h2 style="margin:0;">Pasaje Emitido - %s</h2>
				<p style="margin: 5px 0 0 0; opacity: 0.9;">Concepto: %s</p>
			</div>
			<div style="padding: 20px; border: 1px solid #ddd; border-top: none; border-radius: 0 0 5px 5px;">
				<p>Se comunica a <strong>%s</strong>,</p>
				<p>Su pasaje correspondiente a la solicitud <strong>%s</strong> (tramo <strong>%s</strong>) ha sido emitido exitosamente.</p>
				<p>Concepto de la solicitud: <strong>%s</strong></p>

				<h3 style="color: #03738C; border-bottom: 1px solid #eee; padding-bottom: 5px;">Detalles del Vuelo</h3>
				<table style="width: 100%%; border-collapse: collapse; margin-top: 10px;">
					<tr>
						<td style="padding: 8px 0; color: #666; width: 35%%;"><strong>Tramo:</strong></td>
						<td style="padding: 8px 0;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #666;"><strong>Ruta:</strong></td>
						<td style="padding: 8px 0;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #666;"><strong>Fecha y Hora:</strong></td>
						<td style="padding: 8px 0;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #666;"><strong>Empresa de Vuelo:</strong></td>
						<td style="padding: 8px 0;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #666;"><strong>Número de Vuelo:</strong></td>
						<td style="padding: 8px 0;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #666;"><strong>Nº de Billete:</strong></td>
						<td style="padding: 8px 0;">%s</td>
					</tr>
				</table>

				<div style="margin-top: 25px; text-align: center;">
					<a href="%s" target="_blank" style="background-color: #03738C; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold;">Ver Pasaje en el Sistema</a>
				</div>
				<p style="font-size: 12px; color: #999; margin-top: 20px; text-align: center;">
					Si tiene problemas con el enlace, copie y pegue la siguiente URL en su navegador:<br>
					<a href="%s" style="color: #03738C;">%s</a>
				</p>
			</div>
		</div>
	`, concepto, strings.ToUpper(concepto), usuario.GetNombreCompleto(), sol.Codigo, tipoTramoStr, concepto, tipoTramoStr, ruta, fecha, aerolinea, vuelo, billete, fileURL, fileURL, fileURL)

	_ = s.emailService.SendEmail(to, cc, nil, subject, body)
}

// ReversionEmailJob encapsula la tarea de enviar un correo de reversión.
type ReversionEmailJob struct {
	Service  *PasajeService
	PasajeID string
}

func (j *ReversionEmailJob) Name() string {
	return "ReversionEmailJob:" + j.PasajeID
}

func (j *ReversionEmailJob) Run(ctx context.Context) error {
	p, err := j.Service.repo.WithContext(ctx).FindByID(ctx, j.PasajeID)
	if err != nil {
		return err
	}

	sol, err := j.Service.solicitudRepo.WithContext(ctx).FindByID(ctx, p.SolicitudID)
	if err != nil || sol == nil {
		return err
	}

	j.Service.sendReversionEmail(sol, p)
	return nil
}

func (s *PasajeService) sendReversionEmail(sol *models.Solicitud, pasaje *models.Pasaje) {
	usuario := sol.Usuario
	if usuario.Email == "" {
		return
	}

	to := []string{usuario.Email}
	var cc []string
	if usuario.Encargado != nil && usuario.Encargado.Email != "" {
		cc = append(cc, usuario.Encargado.Email)
	}

	concepto := sol.GetConceptoNombre()
	tipoTramoStr := "Tramo"
	if pasaje.SolicitudItem != nil {
		tipoTramoStr = string(pasaje.SolicitudItem.Tipo)
	}

	subject := fmt.Sprintf("[%s] Envío de Pasaje Revertido - Solicitud %s", strings.ToUpper(concepto), sol.Codigo)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333; max-width: 600px;">
			<div style="background-color: #f59e0b; color: white; padding: 15px; border-radius: 5px 5px 0 0;">
				<h2 style="margin:0;">Envío de Pasaje Revertido (Corrección)</h2>
				<p style="margin: 5px 0 0 0; opacity: 0.9;">Solicitud: %s</p>
			</div>
			<div style="padding: 20px; border: 1px solid #ddd; border-top: none; border-radius: 0 0 5px 5px;">
				<p>Se comunica a <strong>%s</strong>,</p>
				<p>El envío de su pasaje correspondiente al tramo <strong>%s</strong> ha sido <strong>revertido</strong> por el administrador para realizar correcciones necesarias.</p>
				<p><strong>Por favor, ignore la notificación de emisión anterior si ya la recibió.</strong> Una vez corregidos los datos, recibirá una nueva confirmación con el pasaje rectificado.</p>

				<h3 style="color: #f59e0b; border-bottom: 1px solid #eee; padding-bottom: 5px;">Detalles del Tramo</h3>
				<ul style="list-style: none; padding: 0;">
					<li><strong>Ruta:</strong> %s</li>
					<li><strong>Fecha Programada Original:</strong> %s</li>
				</ul>

				<p style="font-size: 13px; background-color: #fffbeb; padding: 10px; border-left: 4px solid #f59e0b; color: #92400e;">
					Este proceso es normal cuando se detectan errores en el número de billete, la aerolínea o la fecha registrada por la empresa de transportes.
				</p>
			</div>
		</div>
	`, sol.Codigo, usuario.GetNombreCompleto(), tipoTramoStr, pasaje.GetRutaDisplay(), utils.FormatDateTimeLongES(pasaje.FechaVuelo))

	_ = s.emailService.SendEmail(to, cc, nil, subject, body)
}
