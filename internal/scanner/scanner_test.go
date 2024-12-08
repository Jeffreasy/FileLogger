package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"filesystem-logger/internal/models"
)

func TestScanner(t *testing.T) {
	// Create a temporary test directory
	tempDir, err := os.MkdirTemp("", "scanner_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file structure
	testFiles := map[string]int64{
		"small.txt":       100,
		"large.txt":       5 * 1024 * 1024,
		"subdir/test.txt": 200,
	}

	for path, size := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		data := make([]byte, size)
		if err := os.WriteFile(fullPath, data, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", fullPath, err)
		}
	}

	tests := []struct {
		name           string
		config         models.ScanConfig
		expectedFiles  int
		expectedBlocks int
	}{
		{
			name: "Basic scan with size limit",
			config: models.ScanConfig{
				MaxFileSizeMB:       1,
				ScanRecursively:     true,
				ExportBlockedToJSON: true,
			},
			expectedFiles:  5, // root dir (1) + subdir (1) + files (3: small.txt, large.txt, subdir/test.txt)
			expectedBlocks: 1, // large.txt
		},
		{
			name: "Non-recursive scan",
			config: models.ScanConfig{
				MaxFileSizeMB:       1,
				ScanRecursively:     false,
				ExportBlockedToJSON: true,
			},
			expectedFiles:  4, // root dir (1) + subdir (1) + files in root (2: small.txt, large.txt)
			expectedBlocks: 1, // large.txt
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := New(tt.config)
			result, err := scanner.Scan(tempDir)
			if err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			// Debug: Print all found files
			t.Logf("\nFound files in %s:", tt.name)
			for _, file := range result.Files {
				t.Logf("- %s (isDir: %v)", file.Path, file.IsDirectory)
			}

			// Check total number of files found
			if len(result.Files) != tt.expectedFiles {
				t.Errorf("Expected %d files, got %d",
					tt.expectedFiles, len(result.Files))
			}

			// Count blocked files
			blockedCount := 0
			for _, file := range result.Files {
				if file.IsBlocked {
					blockedCount++
				}
			}
			if blockedCount != tt.expectedBlocks {
				t.Errorf("Expected %d blocked files, got %d",
					tt.expectedBlocks, blockedCount)
			}
		})
	}
}

func TestIsFileSizeAllowed(t *testing.T) {
	tests := []struct {
		name        string
		maxSizeMB   int
		fileSize    int64
		shouldAllow bool
	}{
		{
			name:        "Small file under limit",
			maxSizeMB:   1,
			fileSize:    500 * 1024, // 500KB
			shouldAllow: true,
		},
		{
			name:        "File at exact limit",
			maxSizeMB:   1,
			fileSize:    1 * 1024 * 1024,
			shouldAllow: true,
		},
		{
			name:        "File over limit",
			maxSizeMB:   1,
			fileSize:    2 * 1024 * 1024,
			shouldAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := New(models.ScanConfig{MaxFileSizeMB: tt.maxSizeMB})
			allowed := scanner.isFileSizeAllowed(tt.fileSize)
			if allowed != tt.shouldAllow {
				t.Errorf("Expected isFileSizeAllowed to return %v for size %d with limit %d MB",
					tt.shouldAllow, tt.fileSize, tt.maxSizeMB)
			}
		})
	}
}

// TestFileTypeDetection test de detectie van verschillende bestandstypes
func TestFileTypeDetection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "filetype_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Maak test bestanden met verschillende types
	testFiles := map[string][]byte{
		"text.txt":     []byte("Hello, World!"),
		"image.jpg":    {0xFF, 0xD8, 0xFF, 0xE0}, // JPEG magic numbers
		"document.pdf": {0x25, 0x50, 0x44, 0x46}, // PDF magic numbers
	}

	for name, content := range testFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	scanner := New(models.ScanConfig{
		MaxFileSizeMB:   10,
		AllowedTypes:    []string{".txt", ".pdf"},
		ScanRecursively: true,
	})

	result, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Verify file types
	for _, file := range result.Files {
		if file.IsDirectory {
			continue
		}

		ext := filepath.Ext(file.Name)
		switch ext {
		case ".jpg":
			if !file.IsBlocked {
				t.Errorf("Expected .jpg to be blocked (not in allowed types)")
			}
		case ".txt", ".pdf":
			if file.IsBlocked {
				t.Errorf("Expected %s to be allowed", ext)
			}
		}
	}
}

// TestPatternFiltering test het filteren op basis van patterns
func TestPatternFiltering(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pattern_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	testFiles := []string{
		"normal.txt",
		"test.log",
		".hidden",
		"temp.tmp",
	}

	for _, name := range testFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	scanner := New(models.ScanConfig{
		MaxFileSizeMB:       10,
		BlockedPatterns:     []string{"*.tmp", ".*"},
		ScanRecursively:     true,
		ExportBlockedToJSON: true,
	})

	result, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	for _, file := range result.Files {
		if file.IsDirectory {
			continue
		}

		switch file.Name {
		case "temp.tmp", ".hidden":
			if !file.IsBlocked {
				t.Errorf("Expected %s to be blocked by pattern", file.Name)
			}
		case "normal.txt", "test.log":
			if file.IsBlocked {
				t.Errorf("Expected %s to be allowed", file.Name)
			}
		}
	}
}

// TestInvalidPaths test de foutafhandeling voor ongeldige paden
func TestInvalidPaths(t *testing.T) {
	scanner := New(models.ScanConfig{MaxFileSizeMB: 10})

	tests := []struct {
		name     string
		path     string
		wantErr  bool
		errCheck func(error) bool
	}{
		{
			name:    "Non-existent path",
			path:    "/path/that/does/not/exist",
			wantErr: true,
			errCheck: func(err error) bool {
				return os.IsNotExist(err)
			},
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: true,
			errCheck: func(err error) bool {
				return err != nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := scanner.Scan(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !tt.errCheck(err) {
				t.Errorf("Unexpected error type: %v", err)
			}
		})
	}
}

// TestPermissions test de afhandeling van permissie-gerelateerde scenario's
func TestPermissions(t *testing.T) {
	// Skip on Windows as permission handling works differently
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

	tempDir, err := os.MkdirTemp("", "permission_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a directory with no read permissions
	noReadDir := filepath.Join(tempDir, "no_read")
	if err := os.Mkdir(noReadDir, 0000); err != nil {
		t.Fatalf("Failed to create no-read directory: %v", err)
	}
	defer os.Chmod(noReadDir, 0755) // Herstel permissies voor cleanup

	// On Unix systems, explicitly remove all permissions
	if runtime.GOOS != "windows" {
		if err := os.Chmod(noReadDir, 0000); err != nil {
			t.Fatalf("Failed to remove permissions: %v", err)
		}
	}

	scanner := New(models.ScanConfig{
		MaxFileSizeMB:   10,
		ScanRecursively: true,
	})

	result, err := scanner.Scan(tempDir)
	if err != nil {
		// We expect the scan to continue despite permission errors
		t.Logf("Scan produced error as expected: %v", err)
	}

	// Check if the no-read directory was properly marked
	found := false
	for _, file := range result.Files {
		if file.Path == noReadDir {
			found = true
			if file.AccessError == "" {
				t.Error("Expected access error for no-read directory")
			}
			if !file.IsBlocked {
				t.Error("Expected no-read directory to be blocked")
			}
			if !file.IsDirectory {
				t.Error("Expected no-read path to be marked as directory")
			}
		}
	}
	if !found {
		t.Error("No-read directory not found in scan results")
	}
}
