package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/app"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/database"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/logging"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/telemetry"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--healthcheck" {
		client := &http.Client{Timeout: 2 * time.Second}
		response, err := client.Get("http://127.0.0.1:8080/v1/health/live")
		if err != nil || response.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		_ = response.Body.Close()
		return
	}
	if err := run(); err != nil {
		slog.Error("API stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(cfg.LogLevel)
	slog.SetDefault(logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	shutdownTelemetry, err := telemetry.Configure(ctx, cfg.OTLP.Endpoint, "studio-api", cfg.Environment)
	if err != nil {
		return err
	}
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTelemetry(shutdownContext)
	}()

	application, err := app.New(cfg, logger, pool)
	if err != nil {
		return err
	}
	serverError := make(chan error, 1)
	go func() { serverError <- application.Listen(cfg.HTTPAddress) }()
	logger.Info("API listening", "address", cfg.HTTPAddress, "environment", cfg.Environment, "demo_mode", cfg.DemoMode)

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return application.ShutdownWithContext(shutdownContext)
	case err := <-serverError:
		if errors.Is(err, fiber.ErrServiceUnavailable) {
			return nil
		}
		return err
	}
}
