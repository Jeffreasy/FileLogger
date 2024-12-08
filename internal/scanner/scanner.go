package scanner

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"filesystem-logger/internal/models"
	"filesystem-logger/internal/utils/jsonexport"
)

type Scanner struct {
	config     models.ScanConfig
	progress   *models.ScanProgress
	mu         sync.Mutex
	workChan   chan models.ScanWork
	resultChan chan models.ScanWorkResult
	errorChan  chan error
	doneChan   chan struct{}
	dirWg      sync.WaitGroup
}

func New(config models.ScanConfig) *Scanner {
	if config.WorkerCount <= 0 {
		config.WorkerCount = 4 // default worker count
	}
	if config.BufferSize <= 0 {
		config.BufferSize = 1000 // default buffer size
	}

	return &Scanner{
		config:     config,
		progress:   &models.ScanProgress{StartTime: time.Now()},
		workChan:   make(chan models.ScanWork, config.BufferSize),
		resultChan: make(chan models.ScanWorkResult, config.BufferSize),
		errorChan:  make(chan error, config.BufferSize),
		doneChan:   make(chan struct{}),
	}
}

func (s *Scanner) Scan(root string) (*models.ScanResult, error) {
	if root == "" {
		return nil, fmt.Errorf("empty path provided")
	}

	if _, err := os.Stat(root); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start result collector first
	resultDone := make(chan struct{})
	var result models.ScanResult
	go s.collectResults(&result, resultDone)

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < s.config.WorkerCount; i++ {
		wg.Add(1)
		go s.worker(ctx, &wg)
	}

	// Start initial directory walk
	s.dirWg.Add(1)
	if err := s.startScan(root); err != nil {
		return nil, fmt.Errorf("scan error: %v", err)
	}

	// Create a separate goroutine to close workChan after initial scan
	go func() {
		s.dirWg.Wait()
		close(s.workChan)
	}()

	// Wait for all workers to finish
	wg.Wait()

	// Close result channel and wait for collector to finish
	close(s.resultChan)
	<-resultDone

	result.Duration = time.Since(s.progress.StartTime)
	result.Progress = *s.progress
	result.Success = len(result.Progress.Errors) == 0

	// Export blocked files if configured
	if s.config.ExportBlockedToJSON {
		exportPath := filepath.Join(root, "blocked_files.json")
		if err := jsonexport.ExportBlockedFiles(&result, exportPath); err != nil {
			// Log the error but don't fail the scan
			result.Progress.Errors = append(result.Progress.Errors,
				fmt.Sprintf("Failed to export blocked files: %v", err))
		}
	}

	return &result, nil
}

func (s *Scanner) worker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case work, ok := <-s.workChan:
			if !ok {
				return
			}
			s.processWork(ctx, work)
		}
	}
}

func (s *Scanner) processWork(ctx context.Context, work models.ScanWork) {
	if work.IsDir {
		s.scanDirectory(ctx, work.Path, work.Path)
		return
	}

	fileInfo := models.FileInfo{
		Path: work.Path,
		Name: filepath.Base(work.Path),
	}

	info, err := os.Stat(work.Path)
	if err != nil {
		s.resultChan <- models.ScanWorkResult{FileInfo: fileInfo, Error: err}
		return
	}

	fileInfo.Size = info.Size()
	fileInfo.ModTime = info.ModTime()
	fileInfo.IsDirectory = info.IsDir()
	fileInfo.Extension = strings.ToLower(filepath.Ext(info.Name()))

	if err := s.detectFileType(&fileInfo); err != nil {
		fileInfo.AccessError = err.Error()
	}

	if s.shouldBlockFile(&fileInfo) {
		fileInfo.IsBlocked = true
		fileInfo.BlockReason = s.getBlockReason(&fileInfo)
	}

	atomic.AddInt64(&s.progress.ScannedFiles, 1)
	atomic.AddInt64(&s.progress.ScannedSize, fileInfo.Size)
	if fileInfo.IsBlocked {
		atomic.AddInt64(&s.progress.BlockedFiles, 1)
	}

	s.resultChan <- models.ScanWorkResult{FileInfo: fileInfo}
}

func (s *Scanner) scanDirectory(ctx context.Context, path string, root string) {
	defer s.dirWg.Done()

	// Voeg alleen de root directory toe aan de resultaten
	if path == root {
		dirInfo := models.FileInfo{
			Path:        path,
			Name:        filepath.Base(path),
			IsDirectory: true,
		}
		s.resultChan <- models.ScanWorkResult{FileInfo: dirInfo}
		atomic.AddInt64(&s.progress.TotalFiles, 1)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		s.errorChan <- fmt.Errorf("error reading directory %s: %v", path, err)
		return
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		default:
			fullPath := filepath.Join(path, entry.Name())
			info, err := entry.Info()
			if err != nil {
				s.errorChan <- fmt.Errorf("error getting info for %s: %v", fullPath, err)
				continue
			}

			if info.IsDir() {
				if s.config.ScanRecursively {
					// Recursieve modus: we scannen deze directory ook
					s.dirWg.Add(1)
					dirInfo := models.FileInfo{
						Path:        fullPath,
						Name:        info.Name(),
						IsDirectory: true,
					}
					s.resultChan <- models.ScanWorkResult{FileInfo: dirInfo}
					atomic.AddInt64(&s.progress.TotalFiles, 1)
					go s.scanDirectory(ctx, fullPath, root)
				} else {
					// Niet-recursieve modus: toon deze directory wel, maar scan niet verder
					if path == root {
						dirInfo := models.FileInfo{
							Path:        fullPath,
							Name:        info.Name(),
							IsDirectory: true,
						}
						s.resultChan <- models.ScanWorkResult{FileInfo: dirInfo}
						atomic.AddInt64(&s.progress.TotalFiles, 1)
					}
				}
			} else {
				if s.config.ExportBlockedToJSON && filepath.Base(fullPath) == "blocked_files.json" {
					continue
				}
				// Bestanden altijd verwerken in de workChan
				s.workChan <- models.ScanWork{
					Path:     fullPath,
					IsDir:    false,
					Priority: 1,
				}
			}
		}
	}
}

func (s *Scanner) collectResults(result *models.ScanResult, done chan<- struct{}) {
	defer close(done)

	var files []models.FileInfo
	for res := range s.resultChan {
		if res.Error != nil {
			s.mu.Lock()
			s.progress.Errors = append(s.progress.Errors, res.Error.Error())
			s.mu.Unlock()
			continue
		}
		files = append(files, res.FileInfo)
		s.mu.Lock()
		s.progress.LastUpdated = time.Now()
		s.progress.CurrentDirectory = filepath.Dir(res.FileInfo.Path)
		s.mu.Unlock()
	}
	result.Files = files
}

func (s *Scanner) detectFileType(file *models.FileInfo) error {
	// Open file for type detection
	f, err := os.Open(file.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Read first 512 bytes for MIME type detection
	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil && n == 0 {
		return err
	}

	// Detect MIME type
	file.MimeType = http.DetectContentType(buffer[:n])

	// Set FileType based on extension and MIME type
	if file.Extension != "" {
		file.FileType = strings.TrimPrefix(file.Extension, ".")
	} else {
		file.FileType = strings.Split(file.MimeType, "/")[0]
	}

	return nil
}

func (s *Scanner) shouldBlockFile(file *models.FileInfo) bool {
	// Check file size
	if !s.isFileSizeAllowed(file.Size) {
		return true
	}

	// Check if file type is allowed
	if len(s.config.AllowedTypes) > 0 {
		isAllowed := false
		for _, allowedType := range s.config.AllowedTypes {
			if strings.EqualFold(file.Extension, allowedType) {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return true
		}
	}

	// Check blocked patterns
	for _, pattern := range s.config.BlockedPatterns {
		matched, err := filepath.Match(pattern, file.Name)
		if err == nil && matched {
			return true
		}
	}

	return false
}

func (s *Scanner) getBlockReason(file *models.FileInfo) string {
	if !s.isFileSizeAllowed(file.Size) {
		return "File size exceeds limit"
	}

	if len(s.config.AllowedTypes) > 0 {
		isAllowed := false
		for _, allowedType := range s.config.AllowedTypes {
			if strings.EqualFold(file.Extension, allowedType) {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return "File type not allowed"
		}
	}

	for _, pattern := range s.config.BlockedPatterns {
		matched, _ := filepath.Match(pattern, file.Name)
		if matched {
			return fmt.Sprintf("File matches blocked pattern: %s", pattern)
		}
	}

	return "Unknown reason"
}

func (s *Scanner) isFileSizeAllowed(size int64) bool {
	return size <= int64(s.config.MaxFileSizeMB)*1024*1024
}

func (s *Scanner) startScan(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}

	// Queue the root directory
	s.workChan <- models.ScanWork{
		Path:     root,
		IsDir:    info.IsDir(),
		Priority: 1,
	}

	return nil
}

// GetProgress returns a copy of the current progress
func (s *Scanner) GetProgress() *models.ScanProgress {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a deep copy of progress
	progress := &models.ScanProgress{
		TotalFiles:       s.progress.TotalFiles,
		ScannedFiles:     s.progress.ScannedFiles,
		TotalSize:        s.progress.TotalSize,
		ScannedSize:      s.progress.ScannedSize,
		BlockedFiles:     s.progress.BlockedFiles,
		StartTime:        s.progress.StartTime,
		LastUpdated:      s.progress.LastUpdated,
		CurrentDirectory: s.progress.CurrentDirectory,
	}

	// Copy errors slice
	if len(s.progress.Errors) > 0 {
		progress.Errors = make([]string, len(s.progress.Errors))
		copy(progress.Errors, s.progress.Errors)
	}

	return progress
}
