package jsonexport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"filesystem-logger/internal/models"
)

type ExportData struct {
	Timestamp    time.Time         `json:"timestamp"`
	TotalFiles   int64             `json:"totalFiles"`
	BlockedFiles []models.FileInfo `json:"blockedFiles"`
	ScanDuration time.Duration     `json:"scanDuration"`
	BlockedCount int64             `json:"blockedCount"`
	TotalSize    int64             `json:"totalSize"`
	BlockedSize  int64             `json:"blockedSize"`
}

func ExportBlockedFiles(result *models.ScanResult, outputPath string) error {
	// Verzamel geblokkeerde bestanden
	var blockedFiles []models.FileInfo
	var blockedSize int64

	for _, file := range result.Files {
		if file.IsBlocked {
			blockedFiles = append(blockedFiles, file)
			blockedSize += file.Size
		}
	}

	// Maak export data
	exportData := ExportData{
		Timestamp:    time.Now(),
		TotalFiles:   result.Progress.TotalFiles,
		BlockedFiles: blockedFiles,
		ScanDuration: result.Duration,
		BlockedCount: int64(len(blockedFiles)),
		TotalSize:    result.Progress.TotalSize,
		BlockedSize:  blockedSize,
	}

	// Zorg dat de output directory bestaat
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Schrijf naar JSON bestand
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(exportData); err != nil {
		return fmt.Errorf("failed to encode JSON: %v", err)
	}

	return nil
}
