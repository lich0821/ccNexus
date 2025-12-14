package main

import (
    "errors"
    "net/http"
    "os"
    "os/signal"
    "path/filepath"
    "strconv"
    "syscall"

    "github.com/lich0821/ccNexus/internal/config"
    "github.com/lich0821/ccNexus/internal/logger"
    "github.com/lich0821/ccNexus/internal/proxy"
    "github.com/lich0821/ccNexus/internal/storage"
)

func main() {
    dataDir := resolveDataDir()
    if err := os.MkdirAll(dataDir, 0755); err != nil {
        logger.Error("Failed to create data dir %s: %v", dataDir, err)
        os.Exit(1)
    }

    configPath := filepath.Join(dataDir, "config.json")
    statsPath := filepath.Join(dataDir, "stats.json")
    dbPath := os.Getenv("CCNEXUS_DB_PATH")
    if dbPath == "" {
        dbPath = filepath.Join(dataDir, "ccnexus.db")
    }

    if err := storage.MigrateFromJSON(configPath, statsPath, dbPath); err != nil {
        logger.Error("Migration failed: %v", err)
        os.Exit(1)
    }

    sqliteStorage, err := storage.NewSQLiteStorage(dbPath)
    if err != nil {
        logger.Error("Failed to open SQLite storage: %v", err)
        os.Exit(1)
    }
    defer sqliteStorage.Close()

    cfg, err := loadConfig(sqliteStorage)
    if err != nil {
        logger.Error("Unable to load configuration: %v", err)
        os.Exit(1)
    }

    applyEnvOverrides(cfg)
    setLogLevels(cfg.GetLogLevel())

    if err := cfg.Validate(); err != nil {
        logger.Error("Invalid configuration: %v", err)
        os.Exit(1)
    }

    deviceID, err := sqliteStorage.GetOrCreateDeviceID()
    if err != nil {
        logger.Warn("Failed to get device ID: %v, using default", err)
        deviceID = "default"
    }

    statsAdapter := storage.NewStatsStorageAdapter(sqliteStorage)
    p := proxy.New(cfg, statsAdapter, deviceID)

    errCh := make(chan error, 1)
    go func() {
        errCh <- p.Start()
    }()

    logger.Info("ccNexus headless API listening on :%d (data dir: %s, db: %s)", cfg.GetPort(), dataDir, dbPath)

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    select {
    case sig := <-sigCh:
        logger.Info("Received signal %s, shutting down", sig.String())
        if err := p.Stop(); err != nil {
            logger.Warn("Graceful shutdown failed: %v", err)
        }
    case err := <-errCh:
        if err != nil && !errors.Is(err, http.ErrServerClosed) {
            logger.Error("Proxy server stopped with error: %v", err)
            os.Exit(1)
        }
    }

    logger.Info("ccNexus stopped")
}

func resolveDataDir() string {
    if dir := os.Getenv("CCNEXUS_DATA_DIR"); dir != "" {
        return dir
    }
    if home, err := os.UserHomeDir(); err == nil {
        return filepath.Join(home, ".ccNexus")
    }
    return "/data"
}

func loadConfig(sqliteStorage *storage.SQLiteStorage) (*config.Config, error) {
    adapter := storage.NewConfigStorageAdapter(sqliteStorage)
    cfg, err := config.LoadFromStorage(adapter)
    if err != nil {
        logger.Warn("Failed to load config from storage, using default: %v", err)
        cfg = config.DefaultConfig()
        if saveErr := cfg.SaveToStorage(adapter); saveErr != nil {
            logger.Warn("Failed to persist default config: %v", saveErr)
        }
    }

    // Seed a default endpoint when none are configured to avoid boot failure
    if len(cfg.Endpoints) == 0 {
        logger.Warn("No endpoints found; seeding a default endpoint")
        cfg.Endpoints = config.DefaultConfig().Endpoints
        if saveErr := cfg.SaveToStorage(adapter); saveErr != nil {
            logger.Warn("Failed to persist seeded endpoint: %v", saveErr)
        }
    }
    return cfg, nil
}

func applyEnvOverrides(cfg *config.Config) {
    if portStr := os.Getenv("CCNEXUS_PORT"); portStr != "" {
        if port, err := strconv.Atoi(portStr); err == nil {
            cfg.UpdatePort(port)
        } else {
            logger.Warn("Invalid CCNEXUS_PORT value %q: %v", portStr, err)
        }
    }

    if levelStr := os.Getenv("CCNEXUS_LOG_LEVEL"); levelStr != "" {
        if level, err := strconv.Atoi(levelStr); err == nil {
            cfg.UpdateLogLevel(level)
        } else {
            logger.Warn("Invalid CCNEXUS_LOG_LEVEL value %q: %v", levelStr, err)
        }
    }
}

func setLogLevels(level int) {
    if level < 0 {
        return
    }
    logger.GetLogger().SetMinLevel(logger.LogLevel(level))
    logger.GetLogger().SetConsoleLevel(logger.LogLevel(level))
}
