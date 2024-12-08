package jsonexport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"filesystem-logger/internal/models"
)

func TestExportBlockedFiles(t *testing.T) {
	// Create test data
	result := &models.ScanResult{
		Files: []models.FileInfo{
			{
				Path:        "/test/file1.txt",
				Name:        "file1.txt",
				Size:        1024,
				IsBlocked:   true,
				BlockReason: "File size exceeds limit",
			},
			{
				Path:      "/test/file2.txt",
				Name:      "file2.txt",
				Size:      512,
				IsBlocked: false,
			},
		},
		Progress: models.ScanProgress{
			TotalFiles:   2,
			BlockedFiles: 1,
			TotalSize:    1536,
		},
		Duration: time.Second * 5,
	}

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "jsonexport_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	outputPath := filepath.Join(tempDir, "blocked.json")

	// Test export
	if err := ExportBlockedFiles(result, outputPath); err != nil {
		t.Fatalf("ExportBlockedFiles failed: %v", err)
	}

	// Verify exported file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read exported file: %v", err)
	}

	var exported ExportData
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("Failed to parse exported JSON: %v", err)
	}

	// Verify content
	if len(exported.BlockedFiles) != 1 {
		t.Errorf("Expected 1 blocked file, got %d", len(exported.BlockedFiles))
	}

	if exported.BlockedCount != 1 {
		t.Errorf("Expected BlockedCount=1, got %d", exported.BlockedCount)
	}

	if exported.BlockedSize != 1024 {
		t.Errorf("Expected BlockedSize=1024, got %d", exported.BlockedSize)
	}
}
