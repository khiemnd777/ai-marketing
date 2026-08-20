package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/database"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/logging"
)

func main() {
	if err := run(); err != nil {
		slog.Error("River migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL, logLevel, err := migrationSettings()
	if err != nil {
		return err
	}
	logger := logging.New(logLevel)
	slog.SetDefault(logger)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), &rivermigrate.Config{Logger: logger})
	if err != nil {
		return fmt.Errorf("create River migrator: %w", err)
	}
	result, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		return fmt.Errorf("apply River migrations: %w", err)
	}
	logger.Info("River migrations applied", "versions", len(result.Versions))
	return nil
}

func migrationSettings() (databaseURL string, logLevel string, err error) {
	databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return "", "", errors.New("DATABASE_URL is required to run River migrations")
	}
	logLevel = strings.TrimSpace(os.Getenv("LOG_LEVEL"))
	if logLevel == "" {
		logLevel = "info"
	}
	return databaseURL, logLevel, nil
}
