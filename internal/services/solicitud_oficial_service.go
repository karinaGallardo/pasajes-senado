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

	"gorm.io/gorm"
)

type SolicitudOficialService struct {
	repo                *repositories.SolicitudRepository
	usuarioRepo         *repositories.UsuarioRepository
	tipoItinRepo        *repositories.TipoItinerarioRepository
	codigoSecuenciaRepo *repositories.CodigoSecuenciaRepository
	notificationService *NotificationService
	baseService         *SolicitudService
}

func NewSolicitudOficialService(
	repo *repositories.SolicitudRepository,
	usuarioRepo *repositories.UsuarioRepository,
	tipoItinRepo *repositories.TipoItinerarioRepository,
	codigoSecuenciaRepo *repositories.CodigoSecuenciaRepository,
	notificationService *NotificationService,
	baseService *SolicitudService,
) *SolicitudOficialService {
	return &SolicitudOficialService{
		repo:                repo,
		usuarioRepo:         usuarioRepo,
		tipoItinRepo:        tipoItinRepo,
		codigoSecuenciaRepo: codigoSecuenciaRepo,
		notificationService: notificationService,
		baseService:         baseService,
	}
}

func (s *SolicitudOficialService) CreateOficial(ctx context.Context, req dtos.CreateSolicitudOficialRequest, currentUser *models.Usuario) (*models.Solicitud, error) {
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
		AerolineaID:           utils.NilIfEmpty(req.AerolineaID),
		EstadoSolicitudCodigo: utils.Ptr("SOLICITADO"),
	}

	// Build Items
	var items []models.SolicitudItem

	// Multi-tramo process
	for i, t := range req.Tramos {
		orig := strings.ToUpper(strings.TrimSpace(t.OrigenIATA))
		dest := strings.ToUpper(strings.TrimSpace(t.DestinoIATA))
		if orig == "" || dest == "" {
			continue
		}

		fSalida, err := utils.ParseDateTime(t.FechaSalida)
		if err != nil {
			return nil, fmt.Errorf("error en fecha del tramo #%d: %w", i+1, err)
		}

		// Create item via factory
		item := models.NewSolicitudItem("", t.Tipo, orig, dest, fSalida)
		items = append(items, *item)
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

	if s.baseService != nil {
		go s.baseService.sendCreationEmail(solicitud)
	}

	return solicitud, nil
}

func (s *SolicitudOficialService) UpdateOficial(ctx context.Context, id string, req dtos.CreateSolicitudOficialRequest) error {
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
			"aerolinea_sugerida_id": utils.NilIfEmpty(req.AerolineaID),
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
			orig := strings.ToUpper(strings.TrimSpace(t.OrigenIATA))
			dest := strings.ToUpper(strings.TrimSpace(t.DestinoIATA))
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
					if existing.CanEdit() {
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
				// Creating a new item via Model helper
				newItem := models.NewSolicitudItem(id, t.Tipo, orig, dest, fSalida)
				if err := tx.Create(newItem).Error; err != nil {
					return err
				}
			}
		}

		// 5. Delete "editable" items that were removed in the UI
		editableIDs := solicitud.GetEditableItemIDs()
		if len(editableIDs) > 0 {
			deleteQuery := tx.Where("id IN ?", editableIDs)
			if len(itemsToKeepIDs) > 0 {
				deleteQuery = deleteQuery.Where("id NOT IN ?", itemsToKeepIDs)
			}
			if err := deleteQuery.Delete(&models.SolicitudItem{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).Preload("Items.Pasajes", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).First(&solicitud, "id = ?", id).Error; err == nil {
			// El estado global se recalcula vía Hooks GORM al momento de Update
			if err := tx.Model(&solicitud).Update("estado_solicitud_codigo", solicitud.GetEstado()).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
