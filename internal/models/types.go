package models

import "time"

// FileInfo represents metadata about a file
type FileInfo struct {
	Path        string    `json:"path"`
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	FileType    string    `json:"fileType"`
	MimeType    string    `json:"mimeType"`
	Extension   string    `json:"extension"`
	ModTime     time.Time `json:"modTime"`
	IsDirectory bool      `json:"isDirectory"`
	IsBlocked   bool      `json:"isBlocked"`
	BlockReason string    `json:"blockReason,omitempty"`
	AccessError string    `json:"accessError,omitempty"`
}

// ScanConfig holds configuration for the file system scanner
type ScanConfig struct {
	MaxFileSizeMB       int      `json:"maxFileSizeMB"`
	AllowedTypes        []string `json:"allowedTypes"`
	BlockedPatterns     []string `json:"blockedPatterns"`
	ScanRecursively     bool     `json:"scanRecursively"`
	ExportBlockedToJSON bool     `json:"exportBlockedToJSON"`
	WorkerCount         int      `json:"workerCount"`
	BufferSize          int      `json:"bufferSize"`
}

// ScanProgress represents the current progress of a scan operation
type ScanProgress struct {
	TotalFiles       int64     `json:"totalFiles"`
	ScannedFiles     int64     `json:"scannedFiles"`
	TotalSize        int64     `json:"totalSize"`
	ScannedSize      int64     `json:"scannedSize"`
	BlockedFiles     int64     `json:"blockedFiles"`
	Errors           []string  `json:"errors"`
	StartTime        time.Time `json:"startTime"`
	LastUpdated      time.Time `json:"lastUpdated"`
	CurrentDirectory string    `json:"currentDirectory"`
}

// ScanResult contains the final results of a scan operation
type ScanResult struct {
	Files    []FileInfo    `json:"files"`
	Progress ScanProgress  `json:"progress"`
	Duration time.Duration `json:"duration"`
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
}

// ScanWork represents a unit of work for the scanner
type ScanWork struct {
	Path     string
	IsDir    bool
	Priority int
}

// ScanWorkResult represents the result of processing a single path
type ScanWorkResult struct {
	FileInfo FileInfo
	Error    error
}
