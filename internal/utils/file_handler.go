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
