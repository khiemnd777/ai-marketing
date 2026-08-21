package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment   string
	AppURL        string
	HTTPAddress   string
	DatabaseURL   string
	VerticalsDir  string
	WorkerTempDir string
	DemoMode      bool
	LogLevel      string

	SessionSecret []byte
	EncryptionKey []byte
	SessionTTL    time.Duration

	OpenAI   OpenAIConfig
	Seedance SeedanceConfig
	R2       R2Config
	Meta     MetaConfig
	Renderer RendererConfig
	OTLP     OTLPConfig
}

type OpenAIConfig struct {
	APIKey             string
	BaseURL            string
	Model              string
	TranscriptionModel string
	ReasoningEffort    string
	Timeout            time.Duration
	InputUSDPer1M      float64
	OutputUSDPer1M     float64
}

type SeedanceConfig struct {
	APIKey        string
	BaseURL       string
	Model         string
	APIVersion    string
	Resolution    string
	AspectRatio   string
	WebhookSecret string
	CallbackURL   string
	Timeout       time.Duration
	PollInterval  time.Duration
	TaskTimeout   time.Duration
	USDPerSecond  float64
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	Endpoint        string
	BrowserEndpoint string
	PublicBaseURL   string
}

type MetaConfig struct {
	AppID         string
	AppSecret     string
	APIVersion    string
	RedirectURL   string
	GraphBaseURL  string
	DialogBaseURL string
}

type RendererConfig struct {
	BaseURL      string
	SharedSecret string
	TempDir      string
}

type OTLPConfig struct {
	Endpoint string
}

func Load() (Config, error) {
	sessionTTL, err := durationHours("SESSION_TTL_HOURS", 12)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Environment:   envOr("APP_ENV", "development"),
		AppURL:        envOr("APP_URL", "http://localhost:3000"),
		HTTPAddress:   envOr("HTTP_ADDRESS", ":8080"),
		DatabaseURL:   strings.TrimSpace(os.Getenv("DATABASE_URL")),
		VerticalsDir:  envOr("VERTICALS_DIR", "../../verticals"),
		WorkerTempDir: envOr("WORKER_TEMP_DIR", "/tmp/studio-worker"),
		LogLevel:      envOr("LOG_LEVEL", "info"),
		SessionTTL:    sessionTTL,
		Renderer: RendererConfig{
			SharedSecret: strings.TrimSpace(os.Getenv("RENDERER_INTERNAL_AUTH_SECRET")),
			TempDir:      envOr("RENDER_TEMP_DIR", "/tmp/studio-renderer"),
		},
		OTLP: OTLPConfig{Endpoint: strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))},
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required to start the API or worker")
	}
	if _, err := url.ParseRequestURI(cfg.AppURL); err != nil {
		return Config{}, fmt.Errorf("APP_URL is invalid: %w", err)
	}
	if cfg.SessionSecret, err = secretBytes("SESSION_SECRET", 32); err != nil {
		return Config{}, err
	}
	if value := strings.TrimSpace(os.Getenv("ENCRYPTION_KEY")); value != "" {
		if cfg.EncryptionKey, err = decodeSecret(value, 32); err != nil {
			return Config{}, fmt.Errorf("ENCRYPTION_KEY: %w", err)
		}
	} else {
		return Config{}, errors.New("ENCRYPTION_KEY is required")
	}
	return cfg, nil
}

func (c Config) SecureCookies() bool {
	return c.Environment == "production" || strings.HasPrefix(c.AppURL, "https://")
}

func (c OpenAIConfig) Validate() error {
	if c.APIKey == "" {
		return errors.New("OpenAI provider is not configured")
	}
	if c.Model == "" || c.BaseURL == "" {
		return errors.New("OpenAI model and base URL are required")
	}
	return nil
}

func (c SeedanceConfig) Validate() error {
	if c.APIKey == "" {
		return errors.New("Seedance provider is not configured")
	}
	if c.BaseURL == "" || c.Model == "" || c.APIVersion == "" || c.Timeout <= 0 || c.PollInterval <= 0 || c.TaskTimeout <= 0 {
		return errors.New("Seedance endpoint, model, API version, and timeouts are required")
	}
	parsed, err := url.Parse(c.BaseURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return errors.New("Seedance base URL must be an absolute HTTPS URL")
	}
	return nil
}

func (c MetaConfig) Validate() error {
	if c.AppID == "" || c.AppSecret == "" || c.APIVersion == "" || c.RedirectURL == "" {
		return errors.New("Meta app ID, app secret, API version, and redirect URL are required")
	}
	for _, raw := range []string{c.GraphBaseURL, c.DialogBaseURL, c.RedirectURL} {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return errors.New("Meta endpoints and redirect URL must be absolute URLs")
		}
	}
	return nil
}

func (c R2Config) Validate() error {
	if c.AccessKeyID == "" || c.SecretAccessKey == "" || c.Bucket == "" || c.Endpoint == "" {
		return errors.New("R2 object storage is not configured")
	}
	return nil
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func boolEnv(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func durationSeconds(name string, fallback int) (time.Duration, error) {
	return durationFromInt(name, fallback, time.Second)
}

func durationHours(name string, fallback int) (time.Duration, error) {
	return durationFromInt(name, fallback, time.Hour)
}

func durationFromInt(name string, fallback int, unit time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return time.Duration(fallback) * unit, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return time.Duration(n) * unit, nil
}

func nonNegativeFloat(name string, fallback float64) (float64, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative number", name)
	}
	return parsed, nil
}

func secretBytes(name string, minimum int) ([]byte, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return nil, fmt.Errorf("%s is required to start the API or worker", name)
	}
	decoded, err := decodeSecret(value, minimum)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return decoded, nil
}

func decodeSecret(value string, minimum int) ([]byte, error) {
	if decoded, err := base64.RawURLEncoding.DecodeString(value); err == nil && len(decoded) >= minimum {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil && len(decoded) >= minimum {
		return decoded, nil
	}
	if len(value) >= minimum {
		return []byte(value), nil
	}
	return nil, fmt.Errorf("must contain at least %d bytes (raw or base64 encoded)", minimum)
}
