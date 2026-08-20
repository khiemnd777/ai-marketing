package providerconfigs_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/cryptox"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/providerconfigs"
)

func TestClientProviderConfigurationIsolationIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pool := providerTestDatabase(t, ctx)
	actorID := uuid.New()
	clientOne, clientTwo := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO internal_users(id,email,display_name,password_hash,role,requires_password_change)VALUES($1,'provider-admin@example.test','Provider Admin','not-used','ADMIN',false)`, actorID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO clients(id,company_name,created_by,updated_by)VALUES($1,'Provider Client One',$3,$3),($2,'Provider Client Two',$3,$3)`, clientOne, clientTwo, actorID); err != nil {
		t.Fatal(err)
	}
	cipher, err := cryptox.New([]byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	service := providerconfigs.NewService(pool, cipher)
	actor := auth.Principal{UserID: actorID, Role: db.InternalUserRoleADMIN}
	metadata := auth.ClientMetadata{RequestID: "provider-config-integration"}
	if _, err = service.SaveMode(ctx, clientOne, providerconfigs.ModeInput{DemoMode: false, Version: 0}, actor, metadata); err != nil {
		t.Fatalf("save client mode: %v", err)
	}
	if _, err = service.SaveMode(ctx, clientOne, providerconfigs.ModeInput{DemoMode: true, Version: 0}, actor, metadata); !errors.Is(err, providerconfigs.ErrConflict) {
		t.Fatalf("stale provider mode write error = %v, want conflict", err)
	}
	secret := "sk-client-one-must-never-be-returned"
	profile, err := service.Save(ctx, clientOne, providerconfigs.OpenAI, providerconfigs.SaveInput{
		Enabled: true,
		Settings: map[string]any{
			"baseUrl": "https://api.openai.com/v1", "model": "gpt-5.6-luna",
			"transcriptionModel": "gpt-4o-mini-transcribe", "reasoningEffort": "medium",
			"timeoutSeconds": 60, "inputUsdPer1M": 1.25, "outputUsdPer1M": 10.0,
		},
		Secrets: map[string]string{"apiKey": secret}, Version: 0,
	}, actor, metadata)
	if err != nil {
		t.Fatalf("save OpenAI config: %v", err)
	}
	if profile.ClientID != clientOne || profile.DemoMode || !profile.Providers[0].Configured {
		t.Fatalf("unexpected client-one profile: %+v", profile)
	}
	serialized, _ := json.Marshal(profile)
	if strings.Contains(string(serialized), secret) {
		t.Fatal("provider secret leaked into the safe response")
	}
	bundle, err := service.Load(ctx, clientOne)
	if err != nil || bundle.OpenAI.APIKey != secret || bundle.OpenAI.Model != "gpt-5.6-luna" {
		t.Fatalf("resolve client-one provider: model=%q secret=%v err=%v", bundle.OpenAI.Model, bundle.OpenAI.APIKey == secret, err)
	}
	other, err := service.Get(ctx, clientTwo)
	if err != nil {
		t.Fatal(err)
	}
	if !other.DemoMode || other.Version != 0 || other.Providers[0].Configured || len(other.Providers[0].ConfiguredSecretFields) != 0 {
		t.Fatalf("client-two observed client-one configuration: %+v", other)
	}
	var ciphertext []byte
	if err = pool.QueryRow(ctx, `SELECT secret_ciphertext FROM provider_configurations WHERE client_id=$1 AND provider='OPENAI'`, clientOne).Scan(&ciphertext); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(ciphertext), secret) {
		t.Fatal("provider secret was stored as plaintext")
	}
}

func providerTestDatabase(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("STUDIO_USE_TESTCONTAINERS") != "true" {
		t.Skip("set STUDIO_USE_TESTCONTAINERS=true to run provider configuration isolation")
	}
	container, err := postgrescontainer.Run(ctx, "postgres:18.4-alpine", postgrescontainer.WithDatabase("studio_provider_integration"), postgrescontainer.WithUsername("studio"), postgrescontainer.WithPassword("studio"), postgrescontainer.BasicWaitStrategies())
	if err != nil {
		t.Fatal(err)
	}
	testcontainers.CleanupContainer(t, container)
	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve repository root")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", "..", "..", ".."))
	migrations, err := filepath.Glob(filepath.Join(root, "database", "migrations", "*.sql"))
	if err != nil || len(migrations) == 0 {
		t.Fatalf("discover migrations: files=%d err=%v", len(migrations), err)
	}
	for _, filename := range migrations {
		contents, readErr := os.ReadFile(filename)
		if readErr != nil {
			t.Fatal(readErr)
		}
		tx, beginErr := pool.Begin(ctx)
		if beginErr != nil {
			t.Fatal(beginErr)
		}
		if _, execErr := tx.Exec(ctx, strings.TrimSpace(string(contents))); execErr != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("apply migration %s: %v", filepath.Base(filename), execErr)
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			t.Fatal(commitErr)
		}
	}
	return pool
}
