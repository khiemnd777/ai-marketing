package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/cryptox"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/database"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/providerconfigs"
)

type options struct {
	clientID        uuid.UUID
	endpoint        string
	browserEndpoint string
	bucket          string
	accessKeyID     string
	secretAccessKey string
}

func main() {
	if err := run(context.Background(), os.Args, os.Getenv); err != nil {
		slog.Error("local object storage configuration failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, getenv func(string) string) error {
	opts, err := loadOptions(args, getenv)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	cipher, err := cryptox.New(cfg.EncryptionKey)
	if err != nil {
		return err
	}

	actor, err := localAdmin(ctx, pool)
	if err != nil {
		return err
	}
	service := providerconfigs.NewService(pool, cipher)
	profile, err := service.Get(ctx, opts.clientID)
	if err != nil {
		return err
	}
	var r2 providerconfigs.ProviderView
	found := false
	for _, provider := range profile.Providers {
		if provider.Provider == providerconfigs.R2 {
			r2, found = provider, true
			break
		}
	}
	if !found {
		return errors.New("R2 provider profile is unavailable")
	}
	if r2.Configured && strings.TrimSpace(settingString(r2.Settings, "accountId")) != "local-minio" {
		return errors.New("R2 is already configured with a non-local profile; refusing to overwrite it with local MinIO")
	}

	updated, err := service.Save(ctx, opts.clientID, providerconfigs.R2, providerconfigs.SaveInput{
		Enabled: true,
		Settings: map[string]any{
			"accountId":       "local-minio",
			"bucket":          opts.bucket,
			"endpoint":        opts.endpoint,
			"browserEndpoint": opts.browserEndpoint,
			"publicBaseUrl":   "",
		},
		Secrets: map[string]string{
			"accessKeyId":     opts.accessKeyID,
			"secretAccessKey": opts.secretAccessKey,
		},
		Version: r2.Version,
	}, actor, auth.ClientMetadata{RequestID: "local-storage-bootstrap", UserAgent: "make configure-local-storage"})
	if err != nil {
		return err
	}
	for _, provider := range updated.Providers {
		if provider.Provider == providerconfigs.R2 && provider.Configured {
			fmt.Printf("configured client-scoped local object storage for %s\n", opts.clientID)
			return nil
		}
	}
	return errors.New("R2 configuration did not become ready")
}

func settingString(settings map[string]any, name string) string {
	value, _ := settings[name].(string)
	return value
}

type queryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func localAdmin(ctx context.Context, pool queryRower) (auth.Principal, error) {
	var actor auth.Principal
	var role string
	err := pool.QueryRow(ctx, `SELECT id,email,display_name,role::text,version FROM internal_users WHERE role='ADMIN' AND status='ACTIVE' ORDER BY created_at LIMIT 1`).Scan(&actor.UserID, &actor.Email, &actor.DisplayName, &role, &actor.Version)
	if err != nil {
		return auth.Principal{}, fmt.Errorf("load active local Admin: %w", err)
	}
	actor.Role = db.InternalUserRole(role)
	return actor, nil
}

func loadOptions(args []string, getenv func(string) string) (options, error) {
	if strings.TrimSpace(getenv("APP_ENV")) != "development" {
		return options{}, errors.New("local object storage bootstrap only runs with APP_ENV=development")
	}
	if len(args) != 2 {
		return options{}, errors.New("usage: configure-local-storage <client-id>")
	}
	clientID, err := uuid.Parse(strings.TrimSpace(args[1]))
	if err != nil || clientID == uuid.Nil {
		return options{}, errors.New("client-id must be a non-zero UUID")
	}
	opts := options{
		clientID:        clientID,
		endpoint:        strings.TrimRight(strings.TrimSpace(getenv("LOCAL_STORAGE_ENDPOINT")), "/"),
		browserEndpoint: strings.TrimRight(strings.TrimSpace(getenv("LOCAL_STORAGE_BROWSER_ENDPOINT")), "/"),
		bucket:          strings.TrimSpace(getenv("LOCAL_STORAGE_BUCKET")),
		accessKeyID:     strings.TrimSpace(getenv("LOCAL_STORAGE_ACCESS_KEY_ID")),
		secretAccessKey: strings.TrimSpace(getenv("LOCAL_STORAGE_SECRET_ACCESS_KEY")),
	}
	if opts.endpoint == "" || opts.browserEndpoint == "" || opts.bucket == "" || opts.accessKeyID == "" || opts.secretAccessKey == "" {
		return options{}, errors.New("LOCAL_STORAGE_ENDPOINT, LOCAL_STORAGE_BROWSER_ENDPOINT, LOCAL_STORAGE_BUCKET, LOCAL_STORAGE_ACCESS_KEY_ID, and LOCAL_STORAGE_SECRET_ACCESS_KEY are required")
	}
	return opts, nil
}
