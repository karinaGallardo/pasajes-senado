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

func (s Solicitud) GetConceptoNombre() string {
	if s.TipoSolicitud != nil && s.TipoSolicitud.ConceptoViaje != nil {
		return s.TipoSolicitud.ConceptoViaje.Nombre
	}
	return ""
}

func (s Solicitud) GetConceptoCodigo() string {
	if s.TipoSolicitud != nil && s.TipoSolicitud.ConceptoViaje != nil {
		return s.TipoSolicitud.ConceptoViaje.Codigo
	}
	return ""
}

func (s *Solicitud) UpdateStatusBasedOnItems() {
	if len(s.Items) == 0 {
		return
	}

	allApproved := true
	allRejected := true
	allFinalized := true
	allEmitidos := true
	hasApproved := false

	for _, item := range s.Items {
		st := item.GetEstado()

		// Consideramos estados de aprobación (Aprobado, Emitido, Finalizado)
		isApp := (st == "APROBADO" || st == "EMITIDO" || st == "FINALIZADO")

		if !isApp {
			allApproved = false
		} else {
			hasApproved = true
		}

		if st != "FINALIZADO" {
			allFinalized = false
		}

		if st != "EMITIDO" && st != "FINALIZADO" {
			allEmitidos = false
		}

		if st != "RECHAZADO" {
			allRejected = false
		}
	}

	newState := "SOLICITADO"

	if allFinalized {
		newState = "FINALIZADO"
	} else if allEmitidos {
		newState = "EMITIDO"
	} else if allApproved {
		newState = "APROBADO"
	} else if hasApproved {
		newState = "PARCIALMENTE_APROBADO"
	} else if allRejected {
		newState = "RECHAZADO"
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

func (s Solicitud) GetRutaSimple() string {
	origen := s.GetOrigenIATA()
	destino := s.GetDestinoIATA()
	if origen == "" || destino == "" {
		return s.GetOrigenCiudad() + " - " + s.GetDestinoCiudad()
	}
	return origen + " - " + destino
}
func (s Solicitud) GetItemIda() *SolicitudItem {
	for i := range s.Items {
		if s.Items[i].Tipo == TipoSolicitudItemIda {
			return &s.Items[i]
		}
	}
	return nil
}

func (s Solicitud) GetItemVuelta() *SolicitudItem {
	for i := range s.Items {
		if s.Items[i].Tipo == TipoSolicitudItemVuelta {
			return &s.Items[i]
		}
	}
	return nil
}
func (s Solicitud) GetMaxFechaVueloEmitida() *time.Time {
	var maxDate *time.Time
	for _, item := range s.Items {
		for _, p := range item.Pasajes {
			if p.GetEstadoCodigo() == "EMITIDO" {
				if maxDate == nil || p.FechaVuelo.After(*maxDate) {
					fecha := p.FechaVuelo
					maxDate = &fecha
				}
			}
		}
	}
	return maxDate
}

func (s Solicitud) GetUltimoVueloFecha() string {
	maxDate := s.GetMaxFechaVueloEmitida()
	if maxDate == nil {
		return "-"
	}
	return maxDate.Format("02/01/2006")
}

// GetDiasRestantesDescargo calcula cuántos días hábiles le quedan para presentar.
// Retorna un número negativo si ya venció.
func (s Solicitud) GetDiasRestantesDescargo() int {
	maxDate := s.GetMaxFechaVueloEmitida()
	if maxDate == nil {
		return 999
	}

	// 1. Obtener fecha límite (8 días hábiles desde el último vuelo)
	// Como no puedo importar utils aquí directamente (circular dependency),
	// haré la lógica simple o asumiré que se calcula fuera.
	// Pero mejor la pongo aquí si puedo o uso una función auxiliar.
	// Nota: No puedo importar utils. Calculemos aquí.

	diasLimite := 0
	limite := *maxDate
	for diasLimite < 8 {
		limite = limite.AddDate(0, 0, 1)
		if limite.Weekday() != time.Saturday && limite.Weekday() != time.Sunday {
			diasLimite++
		}
	}

	// 2. Contar días hábiles desde HOY hasta el límite
	hoy := time.Now().Truncate(24 * time.Hour)
	limiteTrunc := limite.Truncate(24 * time.Hour)

	if hoy.After(limiteTrunc) {
		// Calcular cuántos días hábiles de mora
		mora := 0
		d := limiteTrunc
		for d.Before(hoy) {
			d = d.AddDate(0, 0, 1)
			if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
				mora++
			}
		}
		return -mora
	}

	// Días restantes
	restantes := 0
	d := hoy
	for d.Before(limiteTrunc) {
		d = d.AddDate(0, 0, 1)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			restantes++
		}
	}
	return restantes
}
