package models

import "time"

type Solicitud struct {
	BaseModel
	Codigo    string  `gorm:"size:12;uniqueIndex"`
	UsuarioID string  `gorm:"size:24;not null"`
	Usuario   Usuario `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`

	CupoDerechoItemID *string          `gorm:"size:36;index;default:null"`
	CupoDerechoItem   *CupoDerechoItem `gorm:"foreignKey:CupoDerechoItemID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	TipoSolicitudCodigo string         `gorm:"size:50;not null;index"`
	TipoSolicitud       *TipoSolicitud `gorm:"foreignKey:TipoSolicitudCodigo;references:Codigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	AmbitoViajeCodigo string       `gorm:"size:20;not null;index"`
	AmbitoViaje       *AmbitoViaje `gorm:"foreignKey:AmbitoViajeCodigo;references:Codigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	TipoItinerarioCodigo string          `gorm:"size:20;not null;index"`
	TipoItinerario       *TipoItinerario `gorm:"foreignKey:TipoItinerarioCodigo;references:Codigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	EstadoSolicitudCodigo *string          `gorm:"size:50;index;default:'SOLICITADO'"`
	EstadoSolicitud       *EstadoSolicitud `gorm:"foreignKey:EstadoSolicitudCodigo;references:Codigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	Viaticos []Viatico `gorm:"foreignKey:SolicitudID"`

	Descargo *Descargo `gorm:"foreignKey:SolicitudID"`

	Autorizacion string `gorm:"size:100;index"`

	Motivo string `gorm:"type:text"`

	AerolineaSugerida string `gorm:"size:100;comment:Aerolinea sugerida para todos los tramos"`

	// New Decoupled Items
	Items []SolicitudItem `gorm:"foreignKey:SolicitudID"`
}

func (Solicitud) TableName() string {
	return "solicitudes"
}

func (s Solicitud) GetEstado() string {
	if s.EstadoSolicitudCodigo == nil {
		return "SOLICITADO"
	}
	return *s.EstadoSolicitudCodigo
}

func (s Solicitud) GetEstadoCodigo() string {
	if s.EstadoSolicitudCodigo == nil {
		return ""
	}
	return *s.EstadoSolicitudCodigo
}
func (s *Solicitud) UpdateStatusBasedOnItems() {
	if len(s.Items) == 0 {
		return
	}

	allApproved := true
	allRejected := true
	allFinalized := true
	anyPending := false
	hasApproved := false
	activeCount := 0

	for _, item := range s.Items {
		st := item.GetEstado()
		// Ignoramos cancelados y pendientes (no solicitados aun) para determinar aprobación
		if st == "CANCELADO" || st == "PENDIENTE" {
			continue
		}
		activeCount++

		if st != "APROBADO" && st != "EMITIDO" && st != "FINALIZADO" {
			allApproved = false
		}
		if st != "FINALIZADO" {
			allFinalized = false
		}
		if st != "RECHAZADO" {
			allRejected = false
		}
		if st == "SOLICITADO" {
			anyPending = true
		}
		if st == "APROBADO" || st == "EMITIDO" || st == "FINALIZADO" {
			hasApproved = true
		}
	}

	newState := "SOLICITADO"

	if activeCount == 0 {
		// Todos cancelados
		newState = "RECHAZADO"
	} else if allFinalized {
		newState = "FINALIZADO"
	} else if allRejected {
		newState = "RECHAZADO"
	} else if allApproved {
		newState = "APROBADO"
	} else if anyPending && hasApproved {
		newState = "PARCIALMENTE_APROBADO"
	} else {
		newState = "SOLICITADO"
	}

	s.EstadoSolicitudCodigo = &newState
}

func (s Solicitud) GetFechaIda() *time.Time {
	for i := range s.Items {
		item := &s.Items[i]
		if item.Tipo == TipoSolicitudItemIda && item.Fecha != nil {
			return item.Fecha
		}
	}
	if len(s.Items) > 0 {
		return s.Items[0].Fecha
	}
	return nil
}

func (s Solicitud) GetFechaVuelta() *time.Time {
	for i := range s.Items {
		item := &s.Items[i]
		if item.Tipo == TipoSolicitudItemVuelta && item.Fecha != nil {
			return item.Fecha
		}
	}
	if len(s.Items) > 1 {
		return s.Items[len(s.Items)-1].Fecha
	}
	return nil
}

func (s Solicitud) GetOrigen() *Destino {
	for i := range s.Items {
		if s.Items[i].Tipo == TipoSolicitudItemIda {
			return s.Items[i].Origen
		}
	}
	if len(s.Items) > 0 {
		return s.Items[0].Origen
	}
	return nil
}

func (s Solicitud) GetDestino() *Destino {
	for i := range s.Items {
		if s.Items[i].Tipo == TipoSolicitudItemIda {
			return s.Items[i].Destino
		}
	}
	if len(s.Items) > 0 {
		return s.Items[0].Destino
	}
	return nil
}

func (s Solicitud) GetOrigenCiudad() string {
	obj := s.GetOrigen()
	if obj != nil {
		return obj.Ciudad
	}
	return "-"
}

func (s Solicitud) GetDestinoCiudad() string {
	obj := s.GetDestino()
	if obj != nil {
		return obj.Ciudad
	}
	return "-"
}

func (s Solicitud) GetOrigenIATA() string {
	obj := s.GetOrigen()
	if obj != nil {
		return obj.IATA
	}
	return ""
}

func (s Solicitud) GetDestinoIATA() string {
	obj := s.GetDestino()
	if obj != nil {
		return obj.IATA
	}
	return ""
}
