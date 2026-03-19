package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"time"
)

type DescargoService struct {
	repo             *repositories.DescargoRepository
	solicitudService *SolicitudService
	usuarioService   *UsuarioService
}

func NewDescargoService(
	repo *repositories.DescargoRepository,
	solicitudService *SolicitudService,
	usuarioService *UsuarioService,
) *DescargoService {
	return &DescargoService{
		repo:             repo,
		solicitudService: solicitudService,
		usuarioService:   usuarioService,
	}
}

func (s *DescargoService) Create(ctx context.Context, req dtos.CreateDescargoRequest, userID string, archivoPaths []string, anexoPaths []string) (*models.Descargo, error) {
	solicitud, err := s.solicitudService.GetByID(ctx, req.SolicitudID)
	if err != nil {
		return nil, err
	}

	descargo := &models.Descargo{
		SolicitudID:       req.SolicitudID,
		UsuarioID:         userID,
		Codigo:            solicitud.Codigo,
		FechaPresentacion: time.Now(),
		Observaciones:     req.Observaciones,
		Estado:            models.EstadoDescargoBorrador,
	}
	descargo.CreatedBy = &userID

	// Si es solicitud oficial, crear detalle PV-06
	isOficial := req.NroMemorandum != "" || req.InformeActividades != "" || req.ObjetivoViaje != "" || req.ResultadosViaje != "" || req.ConclusionesRecomendaciones != "" || req.MontoDevolucion > 0

	if isOficial {
		oficial := &models.DescargoOficial{
			NroMemorandum:               req.NroMemorandum,
			ObjetivoViaje:               req.ObjetivoViaje,
			TipoTransporte:              req.TipoTransporte,
			PlacaVehiculo:               req.PlacaVehiculo,
			InformeActividades:          req.InformeActividades,
			ResultadosViaje:             req.ResultadosViaje,
			ConclusionesRecomendaciones: req.ConclusionesRecomendaciones,
			MontoDevolucion:             req.MontoDevolucion,
			NroBoletaDeposito:           req.NroBoletaDeposito,
			DirigidoA:                   req.DirigidoA,
		}

		// Mapear Anexos
		var anexos []models.AnexoDescargo
		for i, path := range anexoPaths {
			if path != "" {
				anexos = append(anexos, models.AnexoDescargo{
					Archivo: path,
					Orden:   i,
				})
			}
		}
		oficial.Anexos = anexos
		descargo.Oficial = oficial
	}

	// Mapear Detalles de Itinerario
	devoBoletos := make(map[string]bool)
	modBoletos := make(map[string]bool)

	// First pass: identify which Boletos are marked for devo/mod
	for i := range req.ItinTipo {
		idx := req.ItinIndex[i]
		boleto := req.ItinBoleto[i]

		isDevo := false
		isMod := false
		for _, dIdx := range req.ItinDevolucion {
			if dIdx == idx {
				isDevo = true
				break
			}
		}
		for _, mIdx := range req.ItinModificacion {
			if mIdx == idx {
				isMod = true
				break
			}
		}

		if isDevo && boleto != "" {
			devoBoletos[boleto] = true
		}
		if isMod && boleto != "" {
			modBoletos[boleto] = true
		}
	}

	var itinDetalles []models.DetalleItinerarioDescargo
	for i := range req.ItinTipo {
		if i < len(req.ItinRuta) && req.ItinRuta[i] != "" {
			var fecha *time.Time
			if i < len(req.ItinFecha) && req.ItinFecha[i] != "" {
				t := utils.ParseDate("2006-01-02", req.ItinFecha[i])
				fecha = &t
			}

			boleto := ""
			if i < len(req.ItinBoleto) {
				boleto = req.ItinBoleto[i]
			}

			paseNumero := ""
			if i < len(req.ItinPaseNumero) {
				paseNumero = req.ItinPaseNumero[i]
			}

			archivoPath := ""
			if i < len(archivoPaths) {
				archivoPath = archivoPaths[i]
			}

			// All scales of the same ticket share the same devo/mod status
			esDevo := devoBoletos[boleto]
			esMod := modBoletos[boleto]

			itinDetalles = append(itinDetalles, models.DetalleItinerarioDescargo{
				Tipo:              models.TipoDetalleItinerario(req.ItinTipo[i]),
				Ruta:              req.ItinRuta[i],
				Fecha:             fecha,
				Boleto:            boleto,
				EsDevolucion:      esDevo,
				EsModificacion:    esMod,
				NumeroPaseAbordo:  paseNumero,
				ArchivoPaseAbordo: archivoPath,
				Orden:             i,
			})
		}
	}
	descargo.DetallesItinerario = itinDetalles

	if err := s.repo.Create(ctx, descargo); err != nil {
		return nil, err
	}

	return descargo, nil
}

func (s *DescargoService) GetBySolicitudID(ctx context.Context, solicitudID string) (*models.Descargo, error) {
	return s.repo.FindBySolicitudID(ctx, solicitudID)
}

func (s *DescargoService) GetByID(ctx context.Context, id string) (*models.Descargo, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *DescargoService) GetAll(ctx context.Context) ([]models.Descargo, error) {
	return s.repo.FindAll(ctx)
}

func (s *DescargoService) GetCountByUserIDs(ctx context.Context, userIDs []string) int64 {
	count, _ := s.repo.FindCountByUserIDs(ctx, userIDs)
	return count
}

func (s *DescargoService) GetPaginated(ctx context.Context, page, limit int, searchTerm string, userIDs []string) (*repositories.PaginatedDescargos, error) {
	return s.repo.FindPaginated(ctx, page, limit, searchTerm, userIDs)
}

// GetPaginatedScoped resuelve la visibilidad según el rol del usuario:
// Admin/Responsable → ve todos los descargos del sistema
// Encargado         → ve los suyos + los de sus senadores asignados
// Cualquier otro   → solo los propios
func (s *DescargoService) GetPaginatedScoped(ctx context.Context, authUser *models.Usuario, page, limit int, searchTerm string) (*repositories.PaginatedDescargos, error) {
	if authUser.IsAdminOrResponsable() {
		return s.repo.FindPaginated(ctx, page, limit, searchTerm, nil)
	}

	ids := []string{authUser.ID}
	if senators, err := s.usuarioService.GetSenatorsByEncargado(ctx, authUser.ID); err == nil {
		for _, sen := range senators {
			ids = append(ids, sen.ID)
		}
	}

	return s.repo.FindPaginated(ctx, page, limit, searchTerm, ids)
}

func (s *DescargoService) Update(ctx context.Context, descargo *models.Descargo) error {
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoService) UpdateFull(ctx context.Context, id string, req dtos.CreateDescargoRequest, userID string, archivoPaths []string, anexoPaths []string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if descargo.Estado != models.EstadoDescargoBorrador && descargo.Estado != models.EstadoDescargoRechazado {
		return fmt.Errorf("el descargo no se puede editar en su estado actual (%s)", descargo.Estado)
	}

	descargo.Observaciones = req.Observaciones
	descargo.Estado = models.EstadoDescargoBorrador // Revert to draft on update
	descargo.UpdatedBy = &userID

	// Mapear Detalle Oficial
	isOficialUpdate := req.NroMemorandum != "" || req.InformeActividades != "" || req.ObjetivoViaje != "" || req.ResultadosViaje != "" || req.ConclusionesRecomendaciones != "" || req.MontoDevolucion > 0

	if isOficialUpdate {
		if descargo.Oficial == nil {
			descargo.Oficial = &models.DescargoOficial{DescargoID: id}
		}
		descargo.Oficial.NroMemorandum = req.NroMemorandum
		descargo.Oficial.ObjetivoViaje = req.ObjetivoViaje
		descargo.Oficial.TipoTransporte = req.TipoTransporte
		descargo.Oficial.PlacaVehiculo = req.PlacaVehiculo
		descargo.Oficial.InformeActividades = req.InformeActividades
		descargo.Oficial.ResultadosViaje = req.ResultadosViaje
		descargo.Oficial.ConclusionesRecomendaciones = req.ConclusionesRecomendaciones
		descargo.Oficial.MontoDevolucion = req.MontoDevolucion
		descargo.Oficial.NroBoletaDeposito = req.NroBoletaDeposito
		descargo.Oficial.DirigidoA = req.DirigidoA

		// Explicitly save the official detail to ensure columns are updated correctly
		if err := s.repo.UpdateOficial(ctx, descargo.Oficial); err != nil {
			return err
		}

		// Mapear Anexos
		if len(anexoPaths) > 0 {
			var anexos []models.AnexoDescargo
			for i, path := range anexoPaths {
				if path != "" {
					anexos = append(anexos, models.AnexoDescargo{
						DescargoOficialID: descargo.Oficial.ID,
						Archivo:           path,
						Orden:             i,
					})
				}
			}
			// Clear existing ones using oficial ID if exists
			if descargo.Oficial.ID != "" {
				if err := s.repo.ClearAnexos(ctx, descargo.Oficial.ID); err != nil {
					return err
				}
			}
			descargo.Oficial.Anexos = anexos
		}
	}

	// Mapear Detalles de Itinerario
	devolucionMap := make(map[string]bool)
	for _, idx := range req.ItinDevolucion {
		devolucionMap[idx] = true
	}

	modificacionMap := make(map[string]bool)
	for _, idx := range req.ItinModificacion {
		modificacionMap[idx] = true
	}

	var itinDetalles []models.DetalleItinerarioDescargo
	for i := range req.ItinTipo {
		if i < len(req.ItinRuta) && req.ItinRuta[i] != "" {
			var fecha *time.Time
			if i < len(req.ItinFecha) && req.ItinFecha[i] != "" {
				t := utils.ParseDate("2006-01-02", req.ItinFecha[i])
				fecha = &t
			}

			boleto := ""
			if i < len(req.ItinBoleto) {
				boleto = req.ItinBoleto[i]
			}

			paseNumero := ""
			if i < len(req.ItinPaseNumero) {
				paseNumero = req.ItinPaseNumero[i]
			}

			archivoPath := ""
			if i < len(archivoPaths) {
				archivoPath = archivoPaths[i]
			}

			esDevo := false
			esMod := false
			if i < len(req.ItinIndex) {
				esDevo = devolucionMap[req.ItinIndex[i]]
				esMod = modificacionMap[req.ItinIndex[i]]
			}

			itinDetalles = append(itinDetalles, models.DetalleItinerarioDescargo{
				DescargoID:        id,
				Tipo:              models.TipoDetalleItinerario(req.ItinTipo[i]),
				Ruta:              req.ItinRuta[i],
				Fecha:             fecha,
				Boleto:            boleto,
				EsDevolucion:      esDevo,
				EsModificacion:    esMod,
				NumeroPaseAbordo:  paseNumero,
				ArchivoPaseAbordo: archivoPath,
				Orden:             i,
			})
		}
	}

	if err := s.repo.ClearDetalles(ctx, id); err != nil {
		return err
	}

	descargo.DetallesItinerario = itinDetalles
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoService) Submit(ctx context.Context, id, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoBorrador && descargo.Estado != models.EstadoDescargoRechazado {
		return fmt.Errorf("el descargo no se puede enviar en su estado actual (%s)", descargo.Estado)
	}

	descargo.Estado = models.EstadoDescargoEnRevision
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	slog.Info("Descargo enviado a revisión", "id", id, "codigo", descargo.Codigo, "user_id", userID)
	return nil
}

func (s *DescargoService) Approve(ctx context.Context, id string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoEnRevision {
		return errors.New("solo se pueden aprobar descargos en revisión")
	}

	descargo.Estado = models.EstadoDescargoAprobado
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	slog.Info("Descargo aprobado", "id", id, "codigo", descargo.Codigo, "user_id", userID)

	if descargo.SolicitudID != "" {
		return s.solicitudService.Finalize(ctx, descargo.SolicitudID)
	}

	return nil
}

func (s *DescargoService) Reject(ctx context.Context, id string, userID string, observaciones string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoEnRevision {
		return errors.New("solo se pueden observar descargos en revisión")
	}

	descargo.Estado = models.EstadoDescargoRechazado
	descargo.Observaciones = observaciones
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	slog.Info("Descargo rechazado (observado)", "id", id, "codigo", descargo.Codigo, "user_id", userID, "observaciones", observaciones)
	return nil
}

func (s *DescargoService) RevertToDraft(ctx context.Context, id string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoAprobado {
		return errors.New("solo se puede revertir un descargo aprobado")
	}

	descargo.Estado = models.EstadoDescargoBorrador
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	slog.Info("Descargo revertido a borrador", "id", id, "codigo", descargo.Codigo, "user_id", userID)

	if descargo.SolicitudID != "" {
		return s.solicitudService.RevertFinalize(ctx, descargo.SolicitudID)
	}

	return nil
}
