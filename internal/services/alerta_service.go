package services

import (
	"context"
	"fmt"
	"log"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type AlertaService struct {
	solicitudRepo *repositories.SolicitudRepository
	descargoRepo  *repositories.DescargoRepository
	emailService  *EmailService
}

type AlertaDescargoJob struct {
	Service *AlertaService
}

func (j *AlertaDescargoJob) Name() string {
	return "AlertaDescargoJob"
}

func (j *AlertaDescargoJob) Run(ctx context.Context) error {
	return j.Service.ProcesarAlertasDescargo(ctx)
}

func NewAlertaService() *AlertaService {
	return &AlertaService{
		solicitudRepo: repositories.NewSolicitudRepository(),
		descargoRepo:  repositories.NewDescargoRepository(),
		emailService:  NewEmailService(),
	}
}

// ProcesarAlertasDescargo revisa las solicitudes que requieren descargo y envía alertas si están próximas al límite.
func (s *AlertaService) ProcesarAlertasDescargo(ctx context.Context) error {
	log.Println("[AlertaService] Iniciando procesamiento de alertas de descargo...")

	// 1. Obtener solicitudes que potencialmente necesitan descargo (EMITIDO o FINALIZADO)
	// Pero que aún NO tienen un descargo registrado.
	solicitudes, err := s.solicitudRepo.WithContext(ctx).FindPendientesDeDescargo()
	if err != nil {
		return fmt.Errorf("error al buscar solicitudes pendientes de descargo: %w", err)
	}

	hoy := time.Now().Truncate(24 * time.Hour)
	alertasEnviadas := 0

	for _, sol := range solicitudes {
		maxVuelo := sol.GetMaxFechaVueloEmitida()
		if maxVuelo == nil {
			continue
		}
		fechaFin := *maxVuelo

		// La fecha límite son 8 días hábiles después del fin del viaje
		fechaLimite := utils.CalcularFechaLimiteDescargo(fechaFin)

		// Iniciar alertas 2 días antes de la fecha límite
		fechaInicioAlerta := fechaLimite.AddDate(0, 0, -2)

		// Alertar si faltan 2 días o menos, o si ya venció (mora)
		if hoy.After(fechaInicioAlerta) || hoy.Equal(fechaInicioAlerta) {
			// Enviar alerta
			if err := s.enviarAlertaDescargoEmail(sol, fechaLimite); err != nil {
				log.Printf("[AlertaService] Error enviando email para solicitud %s: %v", sol.Codigo, err)
			} else {
				alertasEnviadas++
			}
		}
	}

	log.Printf("[AlertaService] Procesamiento finalizado. Alertas enviadas: %d", alertasEnviadas)
	return nil
}

func (s *AlertaService) obtenerFechaFinViaje(sol models.Solicitud) time.Time {
	var lastDate time.Time
	for _, item := range sol.Items {
		if item.Fecha != nil {
			if lastDate.IsZero() || item.Fecha.After(lastDate) {
				lastDate = *item.Fecha
			}
		}
	}
	return lastDate
}

func (s *AlertaService) enviarAlertaDescargoEmail(sol models.Solicitud, fechaLimite time.Time) error {
	beneficiario := sol.Usuario
	if beneficiario.Email == "" {
		return fmt.Errorf("el beneficiario %s no tiene correo electrónico", beneficiario.GetNombreCompleto())
	}

	destinatarios := []string{beneficiario.Email}
	var copias []string

	// Si tiene encargado, enviar con copia
	if beneficiario.Encargado != nil && beneficiario.Encargado.Email != "" {
		copias = append(copias, beneficiario.Encargado.Email)
	}

	// Copia oculta a pasajesgo (o configuración)
	bccEmail := viper.GetString("ALERTA_BCC_EMAIL")
	if bccEmail == "" {
		bccEmail = "pasajesgo@gmail.com" // Placeholder por defecto sugerido por el usuario
	}
	ocultos := []string{bccEmail}

	subject := fmt.Sprintf("[ALERTA] Pendiente de Descargo de Pasajes - %s", sol.Codigo)

	// Construir cuerpo del mensaje
	fechaLimiteStr := fechaLimite.Format("02/01/2006")
	days := sol.GetDiasRestantesDescargo()
	statusText := ""
	statusBadgeColor := "#EAB308" // Yellow/Orange default

	if days < 0 {
		mora := -days
		statusText = fmt.Sprintf("<span style='color: #DC2626; font-weight: bold;'>%d DÍAS MORA</span>", mora)
		statusBadgeColor = "#DC2626" // Red for danger
	} else {
		statusText = fmt.Sprintf("<span style='color: #059669; font-weight: bold;'>%d DÍAS RESTANTES</span>", days)
		statusBadgeColor = "#03738C" // Teal
	}

	// URL directa a la solicitud
	tipoPath := "oficial"
	if strings.HasPrefix(strings.ToUpper(sol.GetConceptoCodigo()), "DERECHO") {
		tipoPath = "derecho"
	}
	directUrl := fmt.Sprintf("%s/solicitudes/%s/%s/detalle", viper.GetString("APP_URL"), tipoPath, sol.ID)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333; max-width: 600px; border: 1px solid #eee; border-radius: 8px; overflow: hidden;">
			<div style="background-color: %s; color: white; padding: 20px;">
				<h2 style="margin: 0;">Recordatorio de Descargo</h2>
				<p style="margin: 5px 0 0 0; opacity: 0.9;">Solicitud: %s | %s</p>
			</div>
			<div style="padding: 25px; line-height: 1.6;">
				<p>Estimado(a) <strong>%s</strong>,</p>
				<p>Le recordamos que tiene pendiente la presentación del <strong>Descargo de Pasajes</strong> correspondiente a su viaje con código de solicitud <strong>%s</strong>.</p>

				<div style="background-color: #F9FAFB; border-left: 4px solid %s; padding: 15px; margin: 20px 0;">
					<p style="margin: 0; font-weight: bold;">Estado del Plazo: %s</p>
					<p style="margin: 5px 0 0 0; font-weight: bold; color: #4B5563;">Fecha Límite de Presentación: %s</p>
					<p style="margin: 10px 0 0 0; font-size: 14px; color: #6B7280;">
						Recuerde que dispone de 8 días hábiles administrativos a partir de la fecha de retorno para formalizar su descargo.
					</p>
				</div>

				<p>Es importante regularizar este trámite para evitar observaciones administrativas y habilitar futuros requerimientos de pasajes.</p>

				<div style="margin-top: 30px; text-align: center;">
					<a href="%s"
					   style="background-color: #03738C; color: white; padding: 12px 25px; text-decoration: none; border-radius: 6px; font-weight: bold; display: inline-block;">
						Ver Detalles de la Solicitud
					</a>
				</div>
			</div>
			<div style="background-color: #F9FAFB; padding: 15px; text-align: center; border-top: 1px solid #eee; font-size: 12px; color: #6B7280;">
				Este es un mensaje automático del Sistema de Gestión de Pasajes - Senado.
			</div>
		</div>
	`, statusBadgeColor, sol.Codigo, statusText, beneficiario.GetNombreCompleto(), sol.Codigo, statusBadgeColor, statusText, fechaLimiteStr, directUrl)

	return s.emailService.SendEmail(destinatarios, copias, ocultos, subject, body)
}
