package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/database"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/logging"
)

func main() {
	if err := run(); err != nil {
		slog.Error("admin bootstrap failed", "error", err)
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
	email := strings.ToLower(strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL")))
	displayName := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_NAME"))
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if email == "" || len(displayName) < 2 || len(password) < 14 {
		return errors.New("BOOTSTRAP_ADMIN_EMAIL, BOOTSTRAP_ADMIN_NAME, and a 14+ character BOOTSTRAP_ADMIN_PASSWORD are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	queries := db.New(pool)
	count, err := queries.CountInternalUsers(ctx)
	if err != nil {
		return fmt.Errorf("count internal users: %w", err)
	}
	if count != 0 {
		return errors.New("bootstrap is disabled after the first internal user exists")
	}
	passwordHash, err := auth.HashPassword(password, auth.DefaultArgon2Params)
	if err != nil {
		return err
	}
	created, err := queries.CreateInternalUser(ctx, db.CreateInternalUserParams{Email: email, DisplayName: displayName, PasswordHash: passwordHash, Role: db.InternalUserRoleADMIN, RequiresPasswordChange: false})
	if err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == "23505" {
			return errors.New("bootstrap admin already exists")
		}
		return fmt.Errorf("create bootstrap admin: %w", err)
	}
	if err := audit.Record(ctx, queries, audit.Event{Action: "internal_user.bootstrap", EntityType: "internal_user", RequestID: "bootstrap-admin", Outcome: "SUCCESS", After: map[string]any{"id": created.ID, "email": created.Email, "role": created.Role}}); err != nil {
		return fmt.Errorf("write bootstrap audit: %w", err)
	}
	logger.Info("bootstrap admin created", "user_id", created.ID, "email", created.Email)
	return nil
}
