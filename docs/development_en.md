# Development Guide

## Prerequisites

- Go 1.22+
- Node.js 18+
- Wails CLI v2

```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Check environment dependencies
wails doctor
```

## Development Mode

```bash
# Install frontend dependencies
cd frontend && npm install && cd ..

# Start development mode (with hot reload)
wails dev
```

## Build for Release

```bash
npm run build           # Current platform
npm run build:prod      # Production optimized
npm run build:windows   # Windows
npm run build:macos     # macOS
npm run build:linux     # Linux
```

Build output is in `build/bin/` directory.

## Project Structure

```
ccNexus/
├── main.go                 # Application entry
├── app.go                  # Core application logic
├── internal/
│   ├── proxy/              # HTTP proxy core
│   ├── transformer/        # API format transformers
│   ├── storage/            # SQLite data storage
│   ├── config/             # Configuration management
│   ├── webdav/             # WebDAV sync
│   ├── logger/             # Logging system
│   └── tray/               # System tray
└── frontend/               # Frontend code
    ├── src/modules/        # Feature modules
    ├── src/i18n/           # Internationalization
    └── src/themes/         # Theme styles
```
