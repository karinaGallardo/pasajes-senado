package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type PasajeService struct {
	repo              *repositories.PasajeRepository
	solicitudRepo     *repositories.SolicitudRepository
	solicitudItemRepo *repositories.SolicitudItemRepository
	emailService      *EmailService
}

func NewPasajeService() *PasajeService {
	return &PasajeService{
		repo:              repositories.NewPasajeRepository(),
		solicitudRepo:     repositories.NewSolicitudRepository(),
		solicitudItemRepo: repositories.NewSolicitudItemRepository(),
		emailService:      NewEmailService(),
	}
}

func (s *PasajeService) Create(ctx context.Context, solicitudID string, req dtos.CreatePasajeRequest, filePath string) (*models.Pasaje, error) {
	if filePath == "" {
		return nil, fmt.Errorf("el documento del pasaje (PDF) es obligatorio")
	}

	costo := utils.ParseFloat(req.Costo)
	fechaVuelo := utils.ParseDate("2006-01-02T15:04", req.FechaVuelo)

	var aerolineaID *string
	if req.AerolineaID != "" {
		aerolineaID = &req.AerolineaID
	}

	status := "REGISTRADO"

	if req.NumeroBoleto != "" {
		existing, _ := s.repo.WithContext(ctx).FindByNumeroBoleto(req.NumeroBoleto)
		if existing != nil && existing.ID != "" {
			return nil, fmt.Errorf("ya existe un pasaje con el número de boleto %s", req.NumeroBoleto)
		}
	}

	// Rule: One active pasaje per item
	if req.SolicitudItemID != "" {
		item, err := s.solicitudItemRepo.FindByID(req.SolicitudItemID)
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
		Ruta:               req.Ruta,
		FechaVuelo:         fechaVuelo,
		CodigoReserva:      req.CodigoReserva,
		NumeroBoleto:       req.NumeroBoleto,
		NumeroFactura:      req.NumeroFactura,
		Glosa:              req.Glosa,
		Costo:              costo,
		Archivo:            filePath,
	}

	if req.FechaEmision != "" {
		fe := utils.ParseDate("2006-01-02", req.FechaEmision)
		pasaje.FechaEmision = &fe
	}

	err := s.repo.RunTransaction(func(repo *repositories.PasajeRepository, tx *gorm.DB) error {
		if err := repo.Create(pasaje); err != nil {
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
	return s.repo.WithContext(ctx).FindBySolicitudID(solicitudID)
}

func (s *PasajeService) Delete(ctx context.Context, id uint) error {
	return s.repo.WithContext(ctx).Delete(id)
}
func (s *PasajeService) GetByID(ctx context.Context, id string) (*models.Pasaje, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *PasajeService) UpdateFromRequest(ctx context.Context, req dtos.UpdatePasajeRequest, archivo string, paseAbordo string) error {
	pasaje, err := s.repo.WithContext(ctx).FindByID(req.ID)
	if err != nil {
		return err
	}

	// Validate duplicate ticket number
	if req.NumeroBoleto != "" {
		existing, _ := s.repo.WithContext(ctx).FindByNumeroBoleto(req.NumeroBoleto)
		if existing != nil && existing.ID != "" && existing.ID != req.ID {
			return fmt.Errorf("ya existe un pasaje con el número de boleto %s", req.NumeroBoleto)
		}
	}

	pasaje.NumeroVuelo = req.NumeroVuelo
	pasaje.Ruta = req.Ruta
	pasaje.NumeroBoleto = req.NumeroBoleto
	pasaje.NumeroFactura = req.NumeroFactura
	pasaje.CodigoReserva = req.CodigoReserva
	pasaje.Glosa = req.Glosa

	if req.AerolineaID != "" {
		pasaje.AerolineaID = &req.AerolineaID
	}

	pasaje.Costo = utils.ParseFloat(req.Costo)

	pasaje.FechaVuelo = utils.ParseDate("2006-01-02T15:04", req.FechaVuelo)

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

	return s.repo.WithContext(ctx).Update(pasaje)
}

func (s *PasajeService) Update(ctx context.Context, pasaje *models.Pasaje) error {
	return s.repo.WithContext(ctx).Update(pasaje)
}

func (s *PasajeService) DevolverPasaje(ctx context.Context, req dtos.DevolverPasajeRequest) error {
	pasaje, err := s.repo.WithContext(ctx).FindByID(req.PasajeID)
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

	return s.repo.WithContext(ctx).Update(pasaje)
}

func (s *PasajeService) UpdateStatus(ctx context.Context, id string, status string, ticketPath string, pasePath string) error {
	pasaje, err := s.repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}

	pasaje.EstadoPasajeCodigo = &status
	if ticketPath != "" {
		pasaje.Archivo = ticketPath
	}
	if pasePath != "" {
		pasaje.ArchivoPaseAbordo = pasePath
	}

	if err := s.repo.WithContext(ctx).Update(pasaje); err != nil {
		return err
	}

	// If Pasaje is EMITIDO, also update Request Item state to EMITIDO
	if status == "EMITIDO" && pasaje.SolicitudItemID != nil {
		s.solicitudItemRepo.UpdateStatus(*pasaje.SolicitudItemID, "EMITIDO")
	}

	// Trigger email if emitted
	if status == "EMITIDO" {
		sol, _ := s.solicitudRepo.FindByID(pasaje.SolicitudID)
		if sol != nil {
			go s.sendEmissionEmail(sol, pasaje)
		}
	}

	if status == "EMITIDO" {
		go func(p *models.Pasaje) {
			ctx := context.Background()
			sol, err := s.solicitudRepo.WithContext(ctx).FindByID(p.SolicitudID)
			if err == nil && sol != nil {
				s.sendEmissionEmail(sol, p)
			}
		}(pasaje)
	}

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

	subject := fmt.Sprintf("Pasaje Emitido - Solicitud %s", sol.Codigo)

	ruta := pasaje.Ruta
	fecha := utils.FormatDateShortES(pasaje.FechaVuelo)
	boleto := pasaje.NumeroBoleto

	baseURL := viper.GetString("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8284"
	}
	// Asegurar que las barras sean forward slashes para URL
	cleanPath := strings.ReplaceAll(pasaje.Archivo, "\\", "/")
	fileURL := fmt.Sprintf("%s/%s", baseURL, cleanPath)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333; max-width: 600px;">
			<div style="background-color: #03738C; color: white; padding: 15px; border-radius: 5px 5px 0 0;">
				<h2 style="margin:0;">Pasaje Emitido</h2>
			</div>
			<div style="padding: 20px; border: 1px solid #ddd; border-top: none; border-radius: 0 0 5px 5px;">
				<p>Estimado/a <strong>%s</strong>,</p>
				<p>Su pasaje correspondiente a la solicitud <strong>%s</strong> ha sido emitido exitosamente.</p>

				<h3 style="color: #03738C; border-bottom: 1px solid #eee; padding-bottom: 5px;">Detalles del Vuelo</h3>
				<table style="width: 100%%; border-collapse: collapse; margin-top: 10px;">
					<tr>
						<td style="padding: 8px 0; color: #666;"><strong>Ruta:</strong></td>
						<td style="padding: 8px 0;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #666;"><strong>Fecha:</strong></td>
						<td style="padding: 8px 0;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #666;"><strong>Boleto:</strong></td>
						<td style="padding: 8px 0;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px 0; color: #666;"><strong>Reserva (PNR):</strong></td>
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
	`, usuario.GetNombreCompleto(), sol.Codigo, ruta, fecha, boleto, pasaje.CodigoReserva, fileURL, fileURL, fileURL)

	_ = s.emailService.SendEmail(to, cc, nil, subject, body)
}
