package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
)

type PasajeService struct {
	repo          *repositories.PasajeRepository
	estadoRepo    *repositories.EstadoPasajeRepository
	solicitudRepo *repositories.SolicitudRepository
	emailService  *EmailService
}

func NewPasajeService() *PasajeService {
	return &PasajeService{
		repo:          repositories.NewPasajeRepository(),
		estadoRepo:    repositories.NewEstadoPasajeRepository(),
		solicitudRepo: repositories.NewSolicitudRepository(),
		emailService:  NewEmailService(),
	}
}

func (s *PasajeService) Create(ctx context.Context, solicitudID string, req dtos.CreatePasajeRequest, filePath string) (*models.Pasaje, error) {
	costo := utils.ParseFloat(req.Costo)
	fechaVuelo := utils.ParseDate("2006-01-02T15:04", req.FechaVuelo)

	var aerolineaID *string
	if req.AerolineaID != "" {
		aerolineaID = &req.AerolineaID
	}

	pasaje := &models.Pasaje{
		SolicitudID:        solicitudID,
		EstadoPasajeCodigo: utils.Ptr("RESERVADO"),
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

	if err := s.repo.WithContext(ctx).Create(pasaje); err != nil {
		return nil, err
	}
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

	if archivo != "" {
		pasaje.Archivo = archivo
	}
	if paseAbordo != "" {
		pasaje.ArchivoPaseAbordo = paseAbordo
	}

	return s.repo.WithContext(ctx).Update(pasaje)
}

func (s *PasajeService) Reprogramar(ctx context.Context, req dtos.ReprogramarPasajeRequest, filePath string) error {
	stateReprog := "REPROGRAMADO"
	newState := "EMITIDO"

	costo := utils.ParseFloat(req.Costo)
	penalidad := utils.ParseFloat(req.CostoPenalidad)
	fecha := utils.ParseDate("2006-01-02T15:04", req.FechaVuelo)

	var aerolineaID *string
	if req.AerolineaID != "" {
		aerolineaID = &req.AerolineaID
	}

	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.PasajeRepository) error {
		pasajeAnterior, err := repoTx.FindByID(req.PasajeAnteriorID)
		if err != nil {
			return err
		}

		pasajeAnterior.EstadoPasajeCodigo = &stateReprog
		if err := repoTx.Update(pasajeAnterior); err != nil {
			return err
		}

		nuevoPasaje := &models.Pasaje{
			SolicitudID:        pasajeAnterior.SolicitudID,
			PasajeAnteriorID:   &req.PasajeAnteriorID,
			EstadoPasajeCodigo: &newState,
			AerolineaID:        aerolineaID,
			AgenciaID:          &req.AgenciaID,
			NumeroVuelo:        req.NumeroVuelo,
			Ruta:               req.Ruta,
			FechaVuelo:         fecha,
			NumeroBoleto:       req.NumeroBoleto,
			Costo:              costo,
			CostoPenalidad:     penalidad,
			Archivo:            filePath,
			Glosa:              req.Glosa,
			NumeroFactura:      req.NumeroFactura,
			CodigoReserva:      req.CodigoReserva,
		}

		return repoTx.Create(nuevoPasaje)
	})
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
	pasaje.EstadoPasajeCodigo = utils.Ptr("DEVUELTO")

	if pasaje.Glosa != "" {
		pasaje.Glosa += " | Devolución: " + req.Glosa
	} else {
		pasaje.Glosa = "Devolución: " + req.Glosa
	}
	pasaje.CostoPenalidad = costoPenalidad

	return s.repo.WithContext(ctx).Update(pasaje)
}

func (s *PasajeService) UpdateStatus(ctx context.Context, pasajeID string, status string, archivoPase string) error {
	pasaje, err := s.repo.WithContext(ctx).FindByID(pasajeID)
	if err != nil {
		return err
	}

	pasaje.EstadoPasajeCodigo = &status
	if status == "VALIDANDO_USO" && archivoPase != "" {
		pasaje.ArchivoPaseAbordo = archivoPase
	}

	if err := s.repo.WithContext(ctx).Update(pasaje); err != nil {
		return err
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
					<a href="#" style="background-color: #03738C; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold;">Ver Pasaje en el Sistema</a>
				</div>
			</div>
		</div>
	`, usuario.GetNombreCompleto(), sol.Codigo, ruta, fecha, boleto, pasaje.CodigoReserva)

	_ = s.emailService.SendEmail(to, cc, nil, subject, body)
}
