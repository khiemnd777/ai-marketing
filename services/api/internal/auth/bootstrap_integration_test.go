package auth_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
)

func TestAdminBootstrapIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pool := bootstrapTestDatabase(t, ctx)
	service, err := auth.NewService(pool, []byte("bootstrap-integration-session-key-32-bytes"), 12*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	required, err := service.BootstrapRequired(ctx)
	if err != nil || !required {
		t.Fatalf("empty database should require bootstrap: required=%v err=%v", required, err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO internal_users(email,display_name,password_hash,role) VALUES('operator@example.test','Existing Operator','not-used','OPERATOR')`); err != nil {
		t.Fatalf("seed non-admin user: %v", err)
	}
	required, err = service.BootstrapRequired(ctx)
	if err != nil || !required {
		t.Fatalf("non-admin users must not close bootstrap: required=%v err=%v", required, err)
	}

	result, err := service.BootstrapAdmin(ctx, auth.BootstrapInput{
		Email: "admin@example.test", DisplayName: "First Admin", Password: "Bootstrap-password-2026!",
	}, auth.ClientMetadata{RequestID: "bootstrap-integration"})
	if err != nil {
		t.Fatalf("bootstrap first admin: %v", err)
	}
	if result.Principal.Role != "ADMIN" || result.Principal.RequiresPasswordChange || result.SessionToken == "" || result.CSRFToken == "" {
		t.Fatalf("unexpected bootstrap result: role=%s passwordChange=%v session=%v csrf=%v", result.Principal.Role, result.Principal.RequiresPasswordChange, result.SessionToken != "", result.CSRFToken != "")
	}
	if _, err = pool.Exec(ctx, `UPDATE internal_users SET status='DISABLED' WHERE role='ADMIN'`); err != nil {
		t.Fatalf("disable admin fixture: %v", err)
	}
	required, err = service.BootstrapRequired(ctx)
	if err != nil || required {
		t.Fatalf("a disabled admin must still close bootstrap: required=%v err=%v", required, err)
	}
	_, err = service.BootstrapAdmin(ctx, auth.BootstrapInput{
		Email: "second@example.test", DisplayName: "Second Admin", Password: "Bootstrap-password-2026!",
	}, auth.ClientMetadata{RequestID: "bootstrap-closed"})
	if !errors.Is(err, auth.ErrBootstrapClosed) {
		t.Fatalf("second bootstrap error=%v, want ErrBootstrapClosed", err)
	}

	if _, err = pool.Exec(ctx, `TRUNCATE TABLE internal_users CASCADE`); err != nil {
		t.Fatalf("reset isolated auth fixture: %v", err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var workers sync.WaitGroup
	for _, email := range []string{"race-one@example.test", "race-two@example.test"} {
		workers.Add(1)
		go func(email string) {
			defer workers.Done()
			<-start
			_, bootstrapErr := service.BootstrapAdmin(ctx, auth.BootstrapInput{
				Email: email, DisplayName: "Concurrent Admin", Password: "Bootstrap-password-2026!",
			}, auth.ClientMetadata{RequestID: "bootstrap-race"})
			results <- bootstrapErr
		}(email)
	}
	close(start)
	workers.Wait()
	close(results)
	succeeded, closed := 0, 0
	for bootstrapErr := range results {
		switch {
		case bootstrapErr == nil:
			succeeded++
		case errors.Is(bootstrapErr, auth.ErrBootstrapClosed):
			closed++
		default:
			t.Fatalf("unexpected concurrent bootstrap error: %v", bootstrapErr)
		}
	}
	var adminCount int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM internal_users WHERE role='ADMIN'`).Scan(&adminCount); err != nil {
		t.Fatal(err)
	}
	if succeeded != 1 || closed != 1 || adminCount != 1 {
		t.Fatalf("concurrent bootstrap succeeded=%d closed=%d admins=%d", succeeded, closed, adminCount)
	}
}

func bootstrapTestDatabase(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("STUDIO_USE_TESTCONTAINERS") != "true" {
		t.Skip("set STUDIO_USE_TESTCONTAINERS=true to run the Admin bootstrap integration workflow")
	}
	container, err := postgrescontainer.Run(
		ctx,
		"postgres:18.4-alpine",
		postgrescontainer.WithDatabase("studio_auth_integration"),
		postgrescontainer.WithUsername("studio"),
		postgrescontainer.WithPassword("studio"),
		postgrescontainer.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL 18 Testcontainer: %v", err)
	}
	testcontainers.CleanupContainer(t, container)
	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("resolve Testcontainer database URL: %v", err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to Testcontainer database: %v", err)
	}
	t.Cleanup(pool.Close)

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve auth integration test location")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", "..", "..", ".."))
	migrations, err := filepath.Glob(filepath.Join(root, "database", "migrations", "*.sql"))
	if err != nil || len(migrations) == 0 {
		t.Fatalf("discover migrations: files=%d err=%v", len(migrations), err)
	}
	for _, filename := range migrations {
		contents, readErr := os.ReadFile(filename)
		if readErr != nil {
			t.Fatalf("read migration %s: %v", filepath.Base(filename), readErr)
		}
		tx, beginErr := pool.Begin(ctx)
		if beginErr != nil {
			t.Fatalf("begin migration %s: %v", filepath.Base(filename), beginErr)
		}
		if _, execErr := tx.Exec(ctx, strings.TrimSpace(string(contents))); execErr != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("apply migration %s: %v", filepath.Base(filename), execErr)
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			t.Fatalf("commit migration %s: %v", filepath.Base(filename), commitErr)
		}
	}
	return pool
}
