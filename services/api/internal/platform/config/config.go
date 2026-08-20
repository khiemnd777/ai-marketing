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
	openAITimeout, err := durationSeconds("OPENAI_TIMEOUT_SECONDS", 60)
	if err != nil {
		return Config{}, err
	}
	sessionTTL, err := durationHours("SESSION_TTL_HOURS", 12)
	if err != nil {
		return Config{}, err
	}
	openAIInputPrice, err := nonNegativeFloat("OPENAI_INPUT_USD_PER_1M", 0)
	if err != nil {
		return Config{}, err
	}
	openAIOutputPrice, err := nonNegativeFloat("OPENAI_OUTPUT_USD_PER_1M", 0)
	if err != nil {
		return Config{}, err
	}
	seedancePrice, err := nonNegativeFloat("SEEDANCE_USD_PER_SECOND", 0)
	if err != nil {
		return Config{}, err
	}
	seedanceTimeout, err := durationSeconds("SEEDANCE_HTTP_TIMEOUT_SECONDS", 30)
	if err != nil {
		return Config{}, err
	}
	seedancePollInterval, err := durationSeconds("SEEDANCE_POLL_INTERVAL_SECONDS", 15)
	if err != nil {
		return Config{}, err
	}
	seedanceTaskTimeout, err := durationSeconds("SEEDANCE_TASK_TIMEOUT_SECONDS", 1800)
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
		DemoMode:      boolEnv("DEMO_MODE", false),
		LogLevel:      envOr("LOG_LEVEL", "info"),
		SessionTTL:    sessionTTL,
		OpenAI: OpenAIConfig{
			APIKey:             strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
			BaseURL:            envOr("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			Model:              envOr("OPENAI_MODEL", "gpt-5.6-luna"),
			TranscriptionModel: envOr("OPENAI_TRANSCRIPTION_MODEL", "gpt-4o-mini-transcribe"),
			ReasoningEffort:    envOr("OPENAI_REASONING_EFFORT", "medium"),
			Timeout:            openAITimeout,
			InputUSDPer1M:      openAIInputPrice,
			OutputUSDPer1M:     openAIOutputPrice,
		},
		Seedance: SeedanceConfig{
			APIKey:        strings.TrimSpace(os.Getenv("BYTEPLUS_MODELARK_API_KEY")),
			BaseURL:       envOr("SEEDANCE_BASE_URL", "https://ark.ap-southeast.bytepluses.com/api"),
			Model:         envOr("SEEDANCE_MODEL", "dreamina-seedance-2-0-260128"),
			APIVersion:    envOr("SEEDANCE_API_VERSION", "v3"),
			Resolution:    envOr("SEEDANCE_RESOLUTION", "720p"),
			AspectRatio:   envOr("SEEDANCE_DEFAULT_RATIO", "9:16"),
			WebhookSecret: strings.TrimSpace(os.Getenv("SEEDANCE_WEBHOOK_SECRET")),
			CallbackURL:   strings.TrimSpace(os.Getenv("SEEDANCE_CALLBACK_URL")),
			Timeout:       seedanceTimeout,
			PollInterval:  seedancePollInterval,
			TaskTimeout:   seedanceTaskTimeout,
			USDPerSecond:  seedancePrice,
		},
		R2: R2Config{
			AccountID:       strings.TrimSpace(os.Getenv("R2_ACCOUNT_ID")),
			AccessKeyID:     strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID")),
			SecretAccessKey: strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY")),
			Bucket:          strings.TrimSpace(os.Getenv("R2_BUCKET")),
			Endpoint:        strings.TrimSpace(os.Getenv("R2_ENDPOINT")),
			PublicBaseURL:   strings.TrimSpace(os.Getenv("R2_PUBLIC_BASE_URL")),
		},
		Meta: MetaConfig{
			AppID:         strings.TrimSpace(os.Getenv("META_APP_ID")),
			AppSecret:     strings.TrimSpace(os.Getenv("META_APP_SECRET")),
			APIVersion:    strings.TrimSpace(os.Getenv("META_GRAPH_API_VERSION")),
			RedirectURL:   strings.TrimSpace(os.Getenv("META_REDIRECT_URL")),
			GraphBaseURL:  envOr("META_GRAPH_BASE_URL", "https://graph.facebook.com"),
			DialogBaseURL: envOr("META_DIALOG_BASE_URL", "https://www.facebook.com"),
		},
		Renderer: RendererConfig{
			BaseURL:      envOr("RENDERER_BASE_URL", "http://renderer:8090"),
			SharedSecret: strings.TrimSpace(os.Getenv("RENDERER_SHARED_SECRET")),
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
