# FileLogger

A high-performance concurrent file system scanner and analyzer with real-time progress tracking.

[![CI](https://github.com/Jeffreasy/FileLogger/actions/workflows/ci.yml/badge.svg)](https://github.com/Jeffreasy/FileLogger/actions/workflows/ci.yml)

## Features

- 🚀 Concurrent file scanning with worker pool
- 📁 Recursive and non-recursive directory traversal
- 🔍 File filtering and blocking based on:
  - Size limits
  - File types
  - Custom patterns
- 📊 Real-time progress tracking
- 🌐 REST API and WebSocket interface
- 📝 JSON export of blocked files
- 🐳 Docker support

## Quick Start

### Using Docker

    # Build and start
    docker-compose up -d

    # View logs
    docker-compose logs -f

### Manual Installation

    # Clone repository
    git clone https://github.com/Jeffreasy/FileLogger.git

    # Build
    go build -o filelogger ./cmd/server

    # Run
    ./filelogger

## API Endpoints

- \POST /api/scan\ - Start a new scan
- \GET /api/status\ - Get scan status
- \WS /api/ws\ - WebSocket for real-time updates

## Configuration

Environment variables:
- \SCAN_WORKER_COUNT\ - Number of concurrent workers (default: 4)
- \SCAN_BUFFER_SIZE\ - Channel buffer size (default: 1000)
- \MAX_FILE_SIZE_MB\ - Maximum file size to process (default: 50)

## Development

    # Run tests
    go test ./... -v

    # Run with hot reload
    go run ./cmd/server

## License

MIT
