package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"filesystem-logger/internal/models"
	"filesystem-logger/internal/scanner"
)

func TestStartScan(t *testing.T) {
	// Setup test directory with files
	testDir := setupTestData(t)
	defer os.RemoveAll(testDir)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  bool
		expectedFiles  int
	}{
		{
			name: "Valid scan request",
			requestBody: map[string]interface{}{
				"path": testDir,
				"config": models.ScanConfig{
					MaxFileSizeMB:       50,
					ScanRecursively:     true,
					ExportBlockedToJSON: true,
				},
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedFiles:  5, // root dir + subdir + 3 files
		},
		{
			name: "Invalid path",
			requestBody: map[string]interface{}{
				"path": "",
				"config": models.ScanConfig{
					MaxFileSizeMB: 50,
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
			expectedFiles:  0,
		},
		{
			name: "Non-recursive scan",
			requestBody: map[string]interface{}{
				"path": testDir,
				"config": models.ScanConfig{
					MaxFileSizeMB:       50,
					ScanRecursively:     false,
					ExportBlockedToJSON: true,
				},
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
			expectedFiles:  4, // root dir + subdir + 2 root files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert request body to JSON
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			// Create test request
			req := httptest.NewRequest("POST", "/api/scan", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rec := httptest.NewRecorder()

			// Handle request
			StartScan(rec, req)

			// Check status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Parse response
			var response map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				if !tt.expectedError {
					t.Errorf("Failed to decode response: %v", err)
				}
				return
			}

			// Check response content
			if !tt.expectedError {
				if status, ok := response["status"].(string); !ok || status != "started" {
					t.Errorf("Expected status 'started', got %v", status)
				}

				// Wait for scan to complete
				time.Sleep(100 * time.Millisecond)

				// Check scan results
				scanMutex.RLock()
				result := scanResults[response["path"].(string)]
				scanMutex.RUnlock()

				if result != nil && len(result.Files) != tt.expectedFiles {
					t.Errorf("Expected %d files, got %d", tt.expectedFiles, len(result.Files))
				}
			}
		})
	}

	// Cleanup
	scanMutex.Lock()
	activeScans = make(map[string]*scanner.Scanner)
	scanResults = make(map[string]*models.ScanResult)
	scanMutex.Unlock()
}

func TestGetStatus(t *testing.T) {
	// Setup test directory with files
	testDir := setupTestData(t)
	defer os.RemoveAll(testDir)

	// Setup test data
	scanMutex.Lock()
	activeScans[testDir] = scanner.New(models.ScanConfig{})
	scanMutex.Unlock()

	tests := []struct {
		name           string
		scanID         string
		expectedStatus int
		expectedState  string
	}{
		{
			name:           "Active scan",
			scanID:         testDir,
			expectedStatus: http.StatusOK,
			expectedState:  "running",
		},
		{
			name:           "Unknown scan",
			scanID:         "non-existent",
			expectedStatus: http.StatusNotFound,
			expectedState:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", "/api/status?id="+tt.scanID, nil)
			rec := httptest.NewRecorder()

			// Handle request
			GetStatus(rec, req)

			// Check status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedState != "" {
				var response map[string]interface{}
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if state, ok := response["state"].(string); !ok || state != tt.expectedState {
					t.Errorf("Expected state %s, got %s", tt.expectedState, state)
				}
			}
		})
	}

	// Cleanup
	scanMutex.Lock()
	activeScans = make(map[string]*scanner.Scanner)
	scanResults = make(map[string]*models.ScanResult)
	scanMutex.Unlock()
}

func TestWebSocketHandler(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(WebSocketHandler))
	defer server.Close()

	// Test WebSocket connection
	t.Run("WebSocket connection", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/ws", nil)
		rec := httptest.NewRecorder()

		WebSocketHandler(rec, req)

		// Check that we get a switching protocols response
		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for non-WebSocket request, got %d",
				http.StatusBadRequest, rec.Code)
		}
	})
}

// Helper function to create test files and directories
func setupTestData(t *testing.T) string {
	t.Helper()

	// Create temporary test directory
	tempDir := t.TempDir()

	// Create test files
	testFiles := map[string][]byte{
		"small.txt":       []byte("Hello, World!"),
		"large.txt":       make([]byte, 5*1024*1024), // 5MB file
		"subdir/test.txt": []byte("Test file in subdirectory"),
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", fullPath, err)
		}
	}

	return tempDir
}
