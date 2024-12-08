package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"filesystem-logger/internal/models"
	"filesystem-logger/internal/scanner"

	"github.com/gorilla/websocket"
)

var (
	activeScans = make(map[string]*scanner.Scanner)
	scanResults = make(map[string]*models.ScanResult)
	scanMutex   sync.RWMutex
)

func StartScan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path   string            `json:"path"`
		Config models.ScanConfig `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "path cannot be empty", http.StatusBadRequest)
		return
	}

	s := scanner.New(req.Config)

	// Start scan in goroutine
	go func() {
		result, err := s.Scan(req.Path)
		if err != nil {
			scanMutex.Lock()
			activeScans[req.Path] = nil
			scanMutex.Unlock()
			return
		}
		// Store both scanner and result
		scanMutex.Lock()
		scanResults[req.Path] = result
		activeScans[req.Path] = s
		scanMutex.Unlock()
	}()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "started",
		"path":   req.Path,
	})
}

func GetStatus(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("id")
	if path == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	scanMutex.RLock()
	scanner, scannerExists := activeScans[path]
	result, resultExists := scanResults[path]
	scanMutex.RUnlock()

	status := struct {
		State    string               `json:"state"`
		Progress *models.ScanProgress `json:"progress,omitempty"`
		Result   *models.ScanResult   `json:"result,omitempty"`
	}{
		State: "unknown",
	}

	if !scannerExists {
		if resultExists {
			status.State = "completed"
			status.Result = result
		} else {
			w.WriteHeader(http.StatusNotFound)
			status.State = "not_found"
		}
	} else if scanner == nil {
		status.State = "error"
	} else {
		status.State = "running"
		status.Progress = scanner.GetProgress()
	}

	json.NewEncoder(w).Encode(status)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // In production, check origin
	},
}

func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()

	// Handle real-time updates
}
