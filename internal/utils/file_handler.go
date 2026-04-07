package utils

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// SaveUploadedFile gestiona la persistencia de archivos subidos, retornando la ruta relativa.
func SaveUploadedFile(c *gin.Context, file *multipart.FileHeader, uploadDir string, prefix string) (string, error) {
	if file == nil {
		return "", nil
	}

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return "", err
		}
	}

	ext := filepath.Ext(file.Filename)
	fileName := fmt.Sprintf("%s%d%s", prefix, time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, fileName)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		return "", err
	}

	return filePath, nil
}

// ExtractDescargoFiles procesa de forma masiva los archivos de pases de abordaje para un descargo.
func ExtractDescargoFiles(c *gin.Context, tramoIDs []string) []string {
	var paths []string
	for _, idRow := range tramoIDs {
		// 1. Verificar si hay un archivo existente (pasa de largo si no hay nada nuevo)
		path := c.PostForm("tramo_archivo_existente_" + idRow)

		// 2. Intentar capturar el archivo nuevo subido para esta fila
		if file, err := c.FormFile("tramo_archivo_" + idRow); err == nil {
			savedPath, err := SaveUploadedFile(c, file, "uploads/pases_abordo", "pase_descargo_"+idRow+"_")
			if err == nil {
				path = savedPath
			}
		}
		paths = append(paths, path)
	}
	return paths
}

// ExtractDescargoAnexos procesa los archivos anexos para un informe oficial PV-06.
func ExtractDescargoAnexos(c *gin.Context, id string) []string {
	var paths []string
	form, _ := c.MultipartForm()
	if form == nil {
		return c.PostFormArray("anexos_existentes[]")
	}

	newAnexos := form.File["anexos[]"]
	existentes := c.PostFormArray("anexos_existentes[]")
	paths = append(paths, existentes...)

	for _, fileHeader := range newAnexos {
		savedPath, err := SaveUploadedFile(c, fileHeader, "uploads/anexos", "anexo_edit_"+id+"_")
		if err == nil {
			paths = append(paths, savedPath)
		}
	}
	return paths
}

// ExtractTerrestreFiles procesa los archivos de comprobantes de transporte terrestre.
func ExtractTerrestreFiles(c *gin.Context, itemIDs []string) []string {
	var paths []string
	for _, idRow := range itemIDs {
		// 1. Verificar si hay un archivo existente
		path := c.PostForm("terrestre_archivo_existente_" + idRow)

		// 2. Intentar capturar el archivo nuevo subido para esta fila
		if file, err := c.FormFile("terrestre_archivo_" + idRow); err == nil {
			savedPath, err := SaveUploadedFile(c, file, "uploads/terrestre", "terrestre_"+idRow+"_")
			if err == nil {
				path = savedPath
			}
		}
		paths = append(paths, path)
	}
	return paths
}
