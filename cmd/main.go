package main

import (
	"fmt"
	"log"

	"filesystem-logger/internal/models"
	"filesystem-logger/internal/scanner"
)

func main() {
	config := models.ScanConfig{
		MaxFileSizeMB:       50,
		ScanRecursively:     true,
		ExportBlockedToJSON: true,
		WorkerCount:         4,
		BufferSize:          1000,
	}

	scanner := scanner.New(config)

	result, err := scanner.Scan("./test-directory")
	if err != nil {
		log.Fatalf("Error scanning directory: %v", err)
	}

	for _, file := range result.Files {
		fmt.Printf("Found: %s, Size: %d bytes\n", file.Path, file.Size)
		if file.IsBlocked {
			fmt.Printf("Blocked: %s, Reason: %s\n", file.Path, file.BlockReason)
		}
	}

	fmt.Printf("\nScan Statistics:\n")
	fmt.Printf("Total Files: %d\n", result.Progress.TotalFiles)
	fmt.Printf("Blocked Files: %d\n", result.Progress.BlockedFiles)
	fmt.Printf("Total Size: %.2f MB\n", float64(result.Progress.TotalSize)/(1024*1024))
	fmt.Printf("Duration: %v\n", result.Duration)
}
