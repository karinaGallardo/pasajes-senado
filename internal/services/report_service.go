package services

import (
	"fmt"
	"os"

	"sistema-pasajes/internal/repositories"

	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"github.com/jung-kurt/gofpdf"
)

type ReportService struct {
	solicitudRepo  *repositories.SolicitudRepository
	aerolineaRepo  *repositories.AerolineaRepository
	pasajeRepo     *repositories.PasajeRepository
	agenciaRepo    *repositories.AgenciaRepository
	cupoRepo       *repositories.CupoDerechoRepository
	openTicketRepo *repositories.OpenTicketRepository
	configService  *ConfiguracionService
}

func NewReportService(
	solicitudRepo *repositories.SolicitudRepository,
	aerolineaRepo *repositories.AerolineaRepository,
	pasajeRepo *repositories.PasajeRepository,
	agenciaRepo *repositories.AgenciaRepository,
	cupoRepo *repositories.CupoDerechoRepository,
	openTicketRepo *repositories.OpenTicketRepository,
	configService *ConfiguracionService,
) *ReportService {
	return &ReportService{
		solicitudRepo:  solicitudRepo,
		aerolineaRepo:  aerolineaRepo,
		pasajeRepo:     pasajeRepo,
		agenciaRepo:    agenciaRepo,
		cupoRepo:       cupoRepo,
		openTicketRepo: openTicketRepo,
		configService:  configService,
	}
}

func (s *ReportService) drawLabelBox(pdf *gofpdf.Fpdf, tr func(string) string, label, value string, wLabel, wBox float64, sameLine bool) {
	h := 6.0
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(wLabel, h, tr(label), "", 0, "R", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	if len(value) > 75 {
		value = value[:72] + "..."
	}
	pdf.CellFormat(wBox, h, "  "+tr(value), "1", 0, "L", false, 0, "")

	if !sameLine {
		pdf.Ln(h + 2)
	}
}

func (s *ReportService) drawReportHeader(pdf *gofpdf.Fpdf, tr func(string) string, formCode, title, subtitle, gestion, code string) {
	// Adjusting based on border starting at X=3 and ending at X=213 (width 210)
	yBase := 8.0

	// 1. Logo Position (Right) - Now clearly inside the frame
	pdf.Image("web/static/img/logo_senado.png", 185, 4, 21, 0, false, "", 0, "")

	// 2. Left Block (Form Code / solicitud Code) - Starting at X=8 (5mm inside border)
	pdf.SetXY(6, yBase)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(45, 7, formCode, "", 2, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(45, 6, code, "", 0, "C", false, 0, "")

	// 3. Central Block (Titles) - Mathematically centered on the page (Letter width ~215.9)
	pdf.SetXY(0, yBase-1)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(215.9, 8, tr(title), "", 2, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(215.9, 5, tr(subtitle), "", 2, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(215.9, 5, tr(gestion), "", 0, "C", false, 0, "")

	// 4. Horizontal Separator - Matching the exact width of top border
	pdf.SetLineWidth(0.3)
	pdf.Line(3, yBase+18, 213, yBase+18) // Aligned with the bottom of the header box

	pdf.SetY(yBase + 24)
}

func (s *ReportService) drawMemoRow(pdf *gofpdf.Fpdf, tr func(string) string, label, value string) {
	h := 7.0
	pdf.SetFillColor(255, 255, 255) // White background
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(30, h, tr(label), "1", 0, "R", true, 0, "")

	pdf.SetFillColor(255, 255, 255)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(160, h, "  "+tr(value), "1", 1, "L", false, 0, "")
}

func (s *ReportService) drawSignatureBlock(pdf *gofpdf.Fpdf, tr func(string) string, y float64, leftLabel, leftName, leftTitle, rightLabel, rightName, rightTitle string) {
	pdf.SetLineWidth(0.2)
	// Left side
	pdf.Line(35, y, 95, y)
	pdf.SetXY(35, y+2)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(60, 4, tr(leftLabel), "", 1, "C", false, 0, "")
	if leftName != "" {
		pdf.SetX(35)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(60, 4, tr(leftName), "", 1, "C", false, 0, "")
	}
	if leftTitle != "" {
		pdf.SetX(35)
		pdf.SetFont("Arial", "I", 6)
		pdf.CellFormat(60, 3, tr(leftTitle), "", 1, "C", false, 0, "")
	}

	// Right side
	pdf.Line(110, y, 185, y)
	pdf.SetXY(110, y+2)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(75, 4, tr(rightLabel), "", 1, "C", false, 0, "")
	if rightName != "" {
		pdf.SetX(110)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(75, 4, tr(rightName), "", 1, "C", false, 0, "")
	}
	if rightTitle != "" {
		pdf.SetX(110)
		pdf.SetFont("Arial", "I", 6)
		pdf.CellFormat(75, 3, tr(rightTitle), "", 1, "C", false, 0, "")
	}
}

func (s *ReportService) drawPageBorder(pdf *gofpdf.Fpdf) {
	pdf.SetLineWidth(0.3)
	pdf.SetDrawColor(0, 0, 0)
	pdf.Rect(3, 3, 210, 265.5, "D")
	pdf.Rect(3, 268.5, 210, 9.3, "D")
}

// isValidPDF verifica si un archivo existe y tiene la cabecera mágica de un PDF (%PDF-)
func (s *ReportService) isValidPDF(filePath string) bool {
	if filePath == "" {
		return false
	}
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	header := make([]byte, 5)
	n, err := f.Read(header)
	if err != nil || n < 5 {
		return false
	}

	return string(header) == "%PDF-"
}

// getValidImage verifica si un archivo es una imagen y lo convierte a PNG si el formato no es soportado por gofpdf (ej: webp, bmp)
func (s *ReportService) getValidImage(filePath string) (string, bool, error) {
	if filePath == "" {
		return "", false, fmt.Errorf("empty path")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	// Detectar formato
	img, format, err := image.Decode(f)
	if err != nil {
		return "", false, fmt.Errorf("decoding error: %w", err)
	}

	// Formatos soportados nativamente por gofpdf
	if format == "jpeg" || format == "png" || format == "gif" {
		return filePath, false, nil
	}

	// Si es otro formato (webp, bmp, etc.), convertir a PNG temporal
	tmpFile, err := os.CreateTemp("", "img_conv_*.png")
	if err != nil {
		return "", false, err
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, img); err != nil {
		return "", false, err
	}

	return tmpFile.Name(), true, nil
}
