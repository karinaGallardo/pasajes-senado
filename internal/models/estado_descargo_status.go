package models

type EstadoDescargoInfo struct {
	Nombre      string
	Descripcion string
	ColorClass  string
	BadgeClass  string
	Icon        string
}

func EstadoDescargoStatusInfo(e EstadoDescargo) EstadoDescargoInfo {
	switch e {
	case EstadoDescargoBorrador:
		return EstadoDescargoInfo{
			Nombre: "Borrador", Descripcion: "El descargo se encuentra en etapa de preparación y edición por el beneficiario.",
			ColorClass: "border-neutral-400", BadgeClass: "bg-neutral-50 text-neutral-600 border-neutral-100",
			Icon: "ph ph-pencil-line",
		}
	case EstadoDescargoEnRevision:
		return EstadoDescargoInfo{
			Nombre: "En Revisión", Descripcion: "El descargo ha sido enviado y está siendo revisado por el área administrativa.",
			ColorClass: "border-warning-400", BadgeClass: "bg-warning-50 text-warning-700 border-warning-100",
			Icon: "ph ph-clock",
		}
	case EstadoDescargoRechazado:
		return EstadoDescargoInfo{
			Nombre: "Observado", Descripcion: "Se han encontrado observaciones en el descargo. Requiere corrección y reenvío.",
			ColorClass: "border-danger-400", BadgeClass: "bg-danger-50 text-danger-700 border-danger-100",
			Icon: "ph ph-warning-circle",
		}
	case EstadoDescargoOpenTicket:
		return EstadoDescargoInfo{
			Nombre: "En Espera (Reprogramación)", Descripcion: "El descargo tiene tramos pendientes de reprogramar (Open Ticket). Debe re-editarse para agregar los nuevos vuelos.",
			ColorClass: "border-secondary-400", BadgeClass: "bg-secondary-50 text-secondary-700 border-secondary-100",
			Icon: "ph ph-calendar-plus",
		}
	case EstadoDescargoEnRevisionOT:
		return EstadoDescargoInfo{
			Nombre: "En Revisión (Utilización)", Descripcion: "Los nuevos tramos han sido registrados y están esperando validación administrativa.",
			ColorClass: "border-indigo-400", BadgeClass: "bg-indigo-50 text-indigo-700 border-indigo-100",
			Icon: "ph ph-magnifying-glass-plus",
		}
	case EstadoDescargoFinalizado:
		return EstadoDescargoInfo{
			Nombre: "Finalizado", Descripcion: "El descargo ha sido validado y aprobado satisfactoriamente.",
			ColorClass: "border-neutral-900", BadgeClass: "bg-neutral-900 text-white border-neutral-800",
			Icon: "ph ph-flag-checkered",
		}
	default:
		return EstadoDescargoInfo{
			Nombre: string(e), Descripcion: "-",
			ColorClass: "border-neutral-200", BadgeClass: "bg-neutral-50 text-neutral-700",
			Icon: "ph ph-info",
		}
	}
}
