package config

import (
	"encoding/base64"
	"testing"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SESSION_SECRET", base64.RawURLEncoding.EncodeToString(make([]byte, 32)))
	if _, err := Load(); err == nil {
		t.Fatal("expected missing database URL to fail")
	}
}

func TestLoadIgnoresProviderEnvironmentVariables(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://studio:studio@localhost/studio")
	t.Setenv("SESSION_SECRET", base64.RawURLEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("ENCRYPTION_KEY", base64.RawURLEncoding.EncodeToString(make([]byte, 32)))
	t.Setenv("OPENAI_API_KEY", "must-not-be-loaded")
	t.Setenv("R2_ACCESS_KEY_ID", "must-not-be-loaded")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OpenAI.Validate() == nil {
		t.Fatal("expected lazy OpenAI validation to report missing configuration")
	}
	if cfg.OpenAI.APIKey != "" || cfg.R2.AccessKeyID != "" {
		t.Fatal("provider configuration must not be loaded from environment variables")
	}
}
