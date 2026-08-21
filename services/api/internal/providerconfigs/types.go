package providerconfigs

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

const (
	OpenAI   = "OPENAI"
	Seedance = "SEEDANCE"
	R2       = "R2"
	Meta     = "META"
	Renderer = "RENDERER"
)

var kinds = []string{OpenAI, Seedance, R2, Meta, Renderer}

var (
	ErrInvalid  = errors.New("invalid provider configuration")
	ErrNotFound = errors.New("provider configuration not found")
	ErrConflict = errors.New("provider configuration version conflict")
)

type Bundle struct {
	ClientID uuid.UUID
	DemoMode bool
	OpenAI   config.OpenAIConfig
	Seedance config.SeedanceConfig
	R2       config.R2Config
	Meta     config.MetaConfig
	Renderer config.RendererConfig
}

type Profile struct {
	ClientID  uuid.UUID      `json:"clientId"`
	DemoMode  bool           `json:"demoMode"`
	Version   int64          `json:"version"`
	Providers []ProviderView `json:"providers"`
}

type ProviderView struct {
	Provider               string         `json:"provider"`
	Enabled                bool           `json:"enabled"`
	Configured             bool           `json:"configured"`
	Settings               map[string]any `json:"settings"`
	ConfiguredSecretFields []string       `json:"configuredSecretFields"`
	Version                int64          `json:"version"`
}

type SaveInput struct {
	Enabled      bool              `json:"enabled"`
	Settings     map[string]any    `json:"settings"`
	Secrets      map[string]string `json:"secrets"`
	ClearSecrets []string          `json:"clearSecrets"`
	Version      int64             `json:"version"`
}

type ModeInput struct {
	DemoMode bool  `json:"demoMode"`
	Version  int64 `json:"version"`
}

type openAISettings struct {
	BaseURL            string  `json:"baseUrl"`
	Model              string  `json:"model"`
	TranscriptionModel string  `json:"transcriptionModel"`
	ReasoningEffort    string  `json:"reasoningEffort"`
	TimeoutSeconds     int     `json:"timeoutSeconds"`
	InputUSDPer1M      float64 `json:"inputUsdPer1M"`
	OutputUSDPer1M     float64 `json:"outputUsdPer1M"`
}

type seedanceSettings struct {
	BaseURL             string  `json:"baseUrl"`
	Model               string  `json:"model"`
	APIVersion          string  `json:"apiVersion"`
	Resolution          string  `json:"resolution"`
	AspectRatio         string  `json:"aspectRatio"`
	CallbackURL         string  `json:"callbackUrl"`
	TimeoutSeconds      int     `json:"timeoutSeconds"`
	PollIntervalSeconds int     `json:"pollIntervalSeconds"`
	TaskTimeoutSeconds  int     `json:"taskTimeoutSeconds"`
	USDPerSecond        float64 `json:"usdPerSecond"`
}

type r2Settings struct {
	AccountID       string `json:"accountId"`
	Bucket          string `json:"bucket"`
	Endpoint        string `json:"endpoint"`
	BrowserEndpoint string `json:"browserEndpoint"`
	PublicBaseURL   string `json:"publicBaseUrl"`
}

type metaSettings struct {
	AppID         string `json:"appId"`
	APIVersion    string `json:"apiVersion"`
	RedirectURL   string `json:"redirectUrl"`
	GraphBaseURL  string `json:"graphBaseUrl"`
	DialogBaseURL string `json:"dialogBaseUrl"`
}

type rendererSettings struct {
	BaseURL string `json:"baseUrl"`
}

func defaultSettings(kind string) map[string]any {
	var value any
	switch kind {
	case OpenAI:
		value = openAISettings{BaseURL: "https://api.openai.com/v1", Model: "gpt-5.6-luna", TranscriptionModel: "gpt-4o-mini-transcribe", ReasoningEffort: "medium", TimeoutSeconds: 60}
	case Seedance:
		value = seedanceSettings{BaseURL: "https://ark.ap-southeast.bytepluses.com/api", Model: "dreamina-seedance-2-0-260128", APIVersion: "v3", Resolution: "720p", AspectRatio: "9:16", TimeoutSeconds: 30, PollIntervalSeconds: 15, TaskTimeoutSeconds: 1800}
	case R2:
		value = r2Settings{}
	case Meta:
		value = metaSettings{GraphBaseURL: "https://graph.facebook.com", DialogBaseURL: "https://www.facebook.com"}
	case Renderer:
		value = rendererSettings{BaseURL: "http://renderer:8090"}
	default:
		return map[string]any{}
	}
	return asMap(value)
}

func secretNames(kind string) []string {
	switch kind {
	case OpenAI:
		return []string{"apiKey"}
	case Seedance:
		return []string{"apiKey", "webhookSecret"}
	case R2:
		return []string{"accessKeyId", "secretAccessKey"}
	case Meta:
		return []string{"appSecret"}
	default:
		return []string{}
	}
}

func normalizeKind(value string) (string, bool) {
	value = strings.ToUpper(strings.TrimSpace(value))
	for _, item := range kinds {
		if value == item {
			return value, true
		}
	}
	return "", false
}

func validateSecrets(kind string, values map[string]string) error {
	allowed := map[string]bool{}
	for _, name := range secretNames(kind) {
		allowed[name] = true
	}
	for name, value := range values {
		if !allowed[name] || len(strings.TrimSpace(value)) > 16_384 {
			return ErrInvalid
		}
	}
	return nil
}

func configuredSecretFields(values map[string]string) []string {
	fields := make([]string, 0, len(values))
	for name, value := range values {
		if strings.TrimSpace(value) != "" {
			fields = append(fields, name)
		}
	}
	sort.Strings(fields)
	return fields
}

func bundleConfiguration(kind string, settings map[string]any, secrets map[string]string, bundle *Bundle) error {
	switch kind {
	case OpenAI:
		var value openAISettings
		if err := decodeMap(settings, &value); err != nil || value.TimeoutSeconds < 1 || value.TimeoutSeconds > 600 || value.InputUSDPer1M < 0 || value.OutputUSDPer1M < 0 || !oneOf(value.ReasoningEffort, "none", "low", "medium", "high", "xhigh") || invalidHTTPURL(value.BaseURL, true) || strings.TrimSpace(value.Model) == "" || strings.TrimSpace(value.TranscriptionModel) == "" {
			return ErrInvalid
		}
		bundle.OpenAI = config.OpenAIConfig{APIKey: strings.TrimSpace(secrets["apiKey"]), BaseURL: strings.TrimRight(value.BaseURL, "/"), Model: strings.TrimSpace(value.Model), TranscriptionModel: strings.TrimSpace(value.TranscriptionModel), ReasoningEffort: value.ReasoningEffort, Timeout: time.Duration(value.TimeoutSeconds) * time.Second, InputUSDPer1M: value.InputUSDPer1M, OutputUSDPer1M: value.OutputUSDPer1M}
	case Seedance:
		var value seedanceSettings
		if err := decodeMap(settings, &value); err != nil || value.TimeoutSeconds < 1 || value.TimeoutSeconds > 600 || value.PollIntervalSeconds < 1 || value.PollIntervalSeconds > 300 || value.TaskTimeoutSeconds < 30 || value.TaskTimeoutSeconds > 7200 || value.USDPerSecond < 0 || invalidHTTPURL(value.BaseURL, true) || (value.CallbackURL != "" && invalidHTTPURL(value.CallbackURL, true)) || strings.TrimSpace(value.Model) == "" || strings.TrimSpace(value.APIVersion) == "" || !oneOf(value.Resolution, "480p", "720p", "1080p", "4k") || !oneOf(value.AspectRatio, "9:16", "16:9", "1:1") {
			return ErrInvalid
		}
		bundle.Seedance = config.SeedanceConfig{APIKey: strings.TrimSpace(secrets["apiKey"]), BaseURL: strings.TrimRight(value.BaseURL, "/"), Model: strings.TrimSpace(value.Model), APIVersion: strings.TrimSpace(value.APIVersion), Resolution: value.Resolution, AspectRatio: value.AspectRatio, WebhookSecret: strings.TrimSpace(secrets["webhookSecret"]), CallbackURL: strings.TrimSpace(value.CallbackURL), Timeout: time.Duration(value.TimeoutSeconds) * time.Second, PollInterval: time.Duration(value.PollIntervalSeconds) * time.Second, TaskTimeout: time.Duration(value.TaskTimeoutSeconds) * time.Second, USDPerSecond: value.USDPerSecond}
	case R2:
		var value r2Settings
		if err := decodeMap(settings, &value); err != nil || invalidHTTPURL(value.Endpoint, false) || (value.BrowserEndpoint != "" && invalidHTTPURL(value.BrowserEndpoint, false)) || (value.PublicBaseURL != "" && invalidHTTPURL(value.PublicBaseURL, false)) || strings.TrimSpace(value.Bucket) == "" {
			return ErrInvalid
		}
		bundle.R2 = config.R2Config{AccountID: strings.TrimSpace(value.AccountID), AccessKeyID: strings.TrimSpace(secrets["accessKeyId"]), SecretAccessKey: strings.TrimSpace(secrets["secretAccessKey"]), Bucket: strings.TrimSpace(value.Bucket), Endpoint: strings.TrimRight(value.Endpoint, "/"), BrowserEndpoint: strings.TrimRight(value.BrowserEndpoint, "/"), PublicBaseURL: strings.TrimRight(value.PublicBaseURL, "/")}
	case Meta:
		var value metaSettings
		if err := decodeMap(settings, &value); err != nil || invalidHTTPURL(value.GraphBaseURL, true) || invalidHTTPURL(value.DialogBaseURL, true) || invalidHTTPURL(value.RedirectURL, false) || strings.TrimSpace(value.AppID) == "" || strings.TrimSpace(value.APIVersion) == "" {
			return ErrInvalid
		}
		bundle.Meta = config.MetaConfig{AppID: strings.TrimSpace(value.AppID), AppSecret: strings.TrimSpace(secrets["appSecret"]), APIVersion: strings.TrimSpace(value.APIVersion), RedirectURL: strings.TrimSpace(value.RedirectURL), GraphBaseURL: strings.TrimRight(value.GraphBaseURL, "/"), DialogBaseURL: strings.TrimRight(value.DialogBaseURL, "/")}
	case Renderer:
		var value rendererSettings
		if err := decodeMap(settings, &value); err != nil || invalidHTTPURL(value.BaseURL, false) {
			return ErrInvalid
		}
		bundle.Renderer = config.RendererConfig{BaseURL: strings.TrimRight(value.BaseURL, "/")}
	default:
		return ErrInvalid
	}
	return nil
}

func validateConfigured(kind string, bundle Bundle) error {
	switch kind {
	case OpenAI:
		if bundle.DemoMode {
			return validateDemoModel(bundle.OpenAI.Model)
		}
		return bundle.OpenAI.Validate()
	case Seedance:
		if bundle.DemoMode {
			return validateDemoModel(bundle.Seedance.Model)
		}
		return bundle.Seedance.Validate()
	case R2:
		return bundle.R2.Validate()
	case Meta:
		if bundle.DemoMode {
			return nil
		}
		return bundle.Meta.Validate()
	case Renderer:
		if strings.TrimSpace(bundle.Renderer.BaseURL) == "" {
			return ErrInvalid
		}
		return nil
	default:
		return ErrInvalid
	}
}

func validateDemoModel(value string) error {
	if strings.TrimSpace(value) == "" {
		return ErrInvalid
	}
	return nil
}

func invalidHTTPURL(value string, requireHTTPS bool) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return true
	}
	if requireHTTPS {
		return parsed.Scheme != "https"
	}
	return parsed.Scheme != "http" && parsed.Scheme != "https"
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func associatedData(clientID uuid.UUID, kind string) string {
	return fmt.Sprintf("provider-config:%s:%s", clientID, kind)
}
