package main

import "testing"

func TestMigrationSettingsOnlyRequireDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", " postgres://studio:studio@localhost:5432/studio ")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("ENCRYPTION_KEY", "")

	databaseURL, logLevel, err := migrationSettings()
	if err != nil {
		t.Fatalf("migrationSettings() error = %v", err)
	}
	if databaseURL != "postgres://studio:studio@localhost:5432/studio" {
		t.Fatalf("databaseURL = %q", databaseURL)
	}
	if logLevel != "info" {
		t.Fatalf("logLevel = %q, want info", logLevel)
	}
}

func TestMigrationSettingsRejectMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", " ")
	if _, _, err := migrationSettings(); err == nil {
		t.Fatal("migrationSettings() error = nil, want missing DATABASE_URL error")
	}
}
