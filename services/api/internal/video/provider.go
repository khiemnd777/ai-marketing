package video

import (
	"context"
	"errors"
	"time"
)

type Reference struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	Role string `json:"role"`
}

type CreateRequest struct {
	Prompt          string      `json:"prompt"`
	References      []Reference `json:"references"`
	Model           string      `json:"model"`
	Resolution      string      `json:"resolution"`
	AspectRatio     string      `json:"aspectRatio"`
	DurationSeconds int32       `json:"durationSeconds"`
	GenerateAudio   bool        `json:"generateAudio"`
	CallbackURL     string      `json:"callbackUrl,omitempty"`
	TimeoutSeconds  int64       `json:"timeoutSeconds"`
}

type Task struct {
	ID                string         `json:"id"`
	Model             string         `json:"model"`
	Status            string         `json:"status"`
	OutputURL         string         `json:"-"`
	ErrorCode         string         `json:"errorCode,omitempty"`
	ErrorMessage      string         `json:"errorMessage,omitempty"`
	UsageTokens       int64          `json:"usageTokens,omitempty"`
	Seed              *int64         `json:"seed,omitempty"`
	FPS               *int32         `json:"fps,omitempty"`
	Resolution        string         `json:"resolution,omitempty"`
	AspectRatio       string         `json:"aspectRatio,omitempty"`
	DurationSeconds   int32          `json:"durationSeconds,omitempty"`
	GenerateAudio     bool           `json:"generateAudio"`
	ProviderRequestID string         `json:"providerRequestId,omitempty"`
	SafeResponse      map[string]any `json:"safeResponse"`
	CreatedAt         *time.Time     `json:"createdAt,omitempty"`
	UpdatedAt         *time.Time     `json:"updatedAt,omitempty"`
}

type Provider interface {
	Create(context.Context, CreateRequest) (Task, error)
	Get(context.Context, string) (Task, error)
	Cancel(context.Context, string) error
}

type ErrorCategory string

const (
	CategoryConfiguration  ErrorCategory = "CONFIGURATION"
	CategoryAuthentication ErrorCategory = "AUTHENTICATION"
	CategoryRateLimit      ErrorCategory = "RATE_LIMIT"
	CategoryModeration     ErrorCategory = "MODERATION"
	CategoryTimeout        ErrorCategory = "TIMEOUT"
	CategoryOutage         ErrorCategory = "PROVIDER_OUTAGE"
	CategoryInvalid        ErrorCategory = "INVALID_REQUEST"
	CategoryNotFound       ErrorCategory = "NOT_FOUND"
)

type ProviderError struct {
	Category  ErrorCategory
	Code      string
	Message   string
	Retryable bool
	Cause     error
}

func (e *ProviderError) Error() string { return e.Message }
func (e *ProviderError) Unwrap() error { return e.Cause }

func AsProviderError(err error) *ProviderError {
	var providerError *ProviderError
	if errors.As(err, &providerError) {
		return providerError
	}
	return &ProviderError{Category: CategoryOutage, Code: "provider_error", Message: "Seedance provider request failed", Retryable: true, Cause: err}
}
