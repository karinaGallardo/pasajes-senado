package services

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"
)

type FormPV5Service struct {
	solicitudService *SolicitudService
	peopleService    *PeopleService
}

func NewFormPV5Service() *FormPV5Service {
	return &FormPV5Service{
		solicitudService: NewSolicitudService(),
		peopleService:    NewPeopleService(),
	}
}

func (s *FormPV5Service) GeneratePV5(ctx context.Context, solicitudID string) ([]byte, string, error) {
	solicitud, err := s.solicitudService.GetByID(ctx, solicitudID)
	if err != nil {
		return nil, "", err
	}

	persona, _ := s.peopleService.GetSenatorDataByCI(ctx, solicitud.Usuario.CI)

	templatePath := "docs/1 pasajes por derecho/2 vacios/PV 5 POR DERECHO DESCARGO.docx"
	if _, err := os.Stat(templatePath); err != nil {
		return nil, "", fmt.Errorf("plantilla no encontrada: %v", err)
	}

	// Read template
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, "", err
	}

	// Prepare replacements
	replacements := make(map[string]string)
	replacements["NOMBRE Y APELLIDOS:"] = "NOMBRE Y APELLIDOS: " + solicitud.Usuario.GetNombreCompleto()
	replacements["C.I.:"] = "C.I.: " + solicitud.Usuario.CI

	depto := ""
	if persona != nil {
		depto = persona.SenadorData.Departamento
	}
	if depto == "" && solicitud.Usuario.Origen != nil {
		depto = solicitud.Usuario.Origen.Ciudad
	}
	replacements["SENADOR POR EL DEPARTAMENTO:"] = "SENADOR POR EL DEPARTAMENTO: " + depto

	// Month of travel
	mes := ""
	var mainDate *time.Time
	for _, it := range solicitud.Items {
		if it.Fecha != nil {
			mainDate = it.Fecha
			break
		}
	}
	if mainDate != nil {
		mes = utils.TranslateMonth(mainDate.Month()) + " " + mainDate.Format("2006")
	}
	replacements["CORRESPONDIENTE AL MES DE:"] = "CORRESPONDIENTE AL MES DE: " + mes

	// Segments are handled sequentially in fillDocx
	var idaItem, vueltaItem *models.SolicitudItem
	for i := range solicitud.Items {
		it := &solicitud.Items[i]
		switch it.Tipo {
		case "IDA":
			idaItem = it
		case "VUELTA":
			vueltaItem = it
		}
	}

	// Process DOCX
	filledDoc, err := s.fillDocx(content, replacements, idaItem, vueltaItem)
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("PV5_Descargo_%s_%s.docx", solicitud.Codigo, solicitud.Usuario.Username)
	return filledDoc, filename, nil
}

func (s *FormPV5Service) fillDocx(template []byte, replacements map[string]string, idaItem, vueltaItem *models.SolicitudItem) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(template), int64(len(template)))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		wf, err := w.Create(f.Name)
		if err != nil {
			rc.Close()
			return nil, err
		}

		if f.Name == "word/document.xml" {
			data, err := io.ReadAll(rc)
			if err != nil {
				rc.Close()
				return nil, err
			}

			// Replacement logic
			filledData := string(data)

			// Section 1: IDA
			if idaItem != nil {
				filledData = strings.Replace(filledData, "RUTA ", "RUTA: "+idaItem.OrigenIATA+" - "+idaItem.DestinoIATA, 1)
				if idaItem.Fecha != nil {
					filledData = strings.Replace(filledData, "FECHA DE VIAJE", "FECHA DE VIAJE: "+idaItem.Fecha.Format("02/01/2006"), 1)
				}
				if len(idaItem.Pasajes) > 0 {
					lastPasaje := idaItem.Pasajes[len(idaItem.Pasajes)-1]
					filledData = strings.Replace(filledData, "N° BOLETO ORIGINAL", "N° BOLETO ORIGINAL: "+lastPasaje.NumeroBoleto, 1)
				}
			}

			// Section 2: RETORNO
			if vueltaItem != nil {
				filledData = strings.Replace(filledData, "RUTA ", "RUTA: "+vueltaItem.OrigenIATA+" - "+vueltaItem.DestinoIATA, 1)
				if vueltaItem.Fecha != nil {
					filledData = strings.Replace(filledData, "FECHA DE VIAJE", "FECHA DE VIAJE: "+vueltaItem.Fecha.Format("02/01/2006"), 1)
				}
				if len(vueltaItem.Pasajes) > 0 {
					lastPasaje := vueltaItem.Pasajes[len(vueltaItem.Pasajes)-1]
					filledData = strings.Replace(filledData, "N° BOLETO ORIGINAL", "N° BOLETO ORIGINAL: "+lastPasaje.NumeroBoleto, 1)
				}
			}

			// Global fields
			for old, new := range replacements {
				filledData = strings.ReplaceAll(filledData, old, new)
			}

			_, err = wf.Write([]byte(filledData))
			if err != nil {
				rc.Close()
				return nil, err
			}
		} else {
			_, err = io.Copy(wf, rc)
			if err != nil {
				rc.Close()
				return nil, err
			}
		}
		rc.Close()
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
