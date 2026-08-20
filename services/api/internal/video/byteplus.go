package video

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

const maxProviderResponseBytes = 2 << 20

type BytePlusProvider struct {
	config config.SeedanceConfig
	client *http.Client
	base   string
}

type bytePlusContent struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *bytePlusURL `json:"image_url,omitempty"`
	VideoURL *bytePlusURL `json:"video_url,omitempty"`
	AudioURL *bytePlusURL `json:"audio_url,omitempty"`
	Role     string       `json:"role,omitempty"`
}
type bytePlusURL struct {
	URL string `json:"url"`
}
type bytePlusCreate struct {
	Model                 string            `json:"model"`
	Content               []bytePlusContent `json:"content"`
	GenerateAudio         bool              `json:"generate_audio"`
	Ratio                 string            `json:"ratio"`
	Duration              int32             `json:"duration"`
	Resolution            string            `json:"resolution"`
	Watermark             bool              `json:"watermark"`
	CallbackURL           string            `json:"callback_url,omitempty"`
	ExecutionExpiresAfter int64             `json:"execution_expires_after,omitempty"`
}
type bytePlusTask struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Status  string `json:"status"`
	Content struct {
		VideoURL string `json:"video_url"`
	} `json:"content"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Usage struct {
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`
	Seed            *int64 `json:"seed"`
	Resolution      string `json:"resolution"`
	Ratio           string `json:"ratio"`
	Duration        int32  `json:"duration"`
	FramesPerSecond *int32 `json:"framespersecond"`
	GenerateAudio   bool   `json:"generate_audio"`
}
type bytePlusError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewBytePlusProvider(cfg config.SeedanceConfig) (*BytePlusProvider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &BytePlusProvider{config: cfg, client: &http.Client{Timeout: cfg.Timeout}, base: strings.TrimRight(cfg.BaseURL, "/") + "/" + strings.Trim(cfg.APIVersion, "/") + "/contents/generations/tasks"}, nil
}

func (p *BytePlusProvider) Create(ctx context.Context, input CreateRequest) (Task, error) {
	if err := validateCreate(input); err != nil {
		return Task{}, err
	}
	content := []bytePlusContent{{Type: "text", Text: input.Prompt}}
	for _, reference := range input.References {
		item := bytePlusContent{Type: reference.Type, Role: reference.Role}
		switch reference.Type {
		case "image_url":
			item.ImageURL = &bytePlusURL{URL: reference.URL}
		case "video_url":
			item.VideoURL = &bytePlusURL{URL: reference.URL}
		case "audio_url":
			item.AudioURL = &bytePlusURL{URL: reference.URL}
		default:
			return Task{}, &ProviderError{Category: CategoryInvalid, Code: "reference_type", Message: "Unsupported Seedance reference type"}
		}
		content = append(content, item)
	}
	payload := bytePlusCreate{Model: input.Model, Content: content, GenerateAudio: input.GenerateAudio, Ratio: input.AspectRatio, Duration: input.DurationSeconds, Resolution: input.Resolution, Watermark: false, CallbackURL: input.CallbackURL, ExecutionExpiresAfter: input.TimeoutSeconds}
	var response bytePlusTask
	requestID, err := p.do(ctx, http.MethodPost, p.base, payload, &response)
	if err != nil {
		return Task{}, err
	}
	if response.ID == "" {
		return Task{}, &ProviderError{Category: CategoryOutage, Code: "empty_task_id", Message: "Seedance returned an empty task ID", Retryable: false}
	}
	return normalizeBytePlus(response, requestID), nil
}

func (p *BytePlusProvider) Get(ctx context.Context, id string) (Task, error) {
	if strings.TrimSpace(id) == "" {
		return Task{}, &ProviderError{Category: CategoryInvalid, Code: "task_id", Message: "Seedance task ID is required"}
	}
	var response bytePlusTask
	requestID, err := p.do(ctx, http.MethodGet, p.base+"/"+url.PathEscape(id), nil, &response)
	if err != nil {
		return Task{}, err
	}
	return normalizeBytePlus(response, requestID), nil
}

func (p *BytePlusProvider) Cancel(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return &ProviderError{Category: CategoryInvalid, Code: "task_id", Message: "Seedance task ID is required"}
	}
	_, err := p.do(ctx, http.MethodDelete, p.base+"/"+url.PathEscape(id), nil, nil)
	return err
}

func (p *BytePlusProvider) do(ctx context.Context, method, endpoint string, payload any, output any) (string, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		category := CategoryOutage
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			category = CategoryTimeout
		}
		return "", &ProviderError{Category: category, Code: "transport_error", Message: "Seedance request did not complete", Retryable: method != http.MethodPost, Cause: err}
	}
	defer response.Body.Close()
	requestID := firstHeader(response.Header, "x-request-id", "x-tt-logid")
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var document bytePlusError
		_ = json.NewDecoder(io.LimitReader(response.Body, maxProviderResponseBytes)).Decode(&document)
		return requestID, classifyHTTPError(response.StatusCode, document.Error.Code)
	}
	if output != nil {
		if err = json.NewDecoder(io.LimitReader(response.Body, maxProviderResponseBytes)).Decode(output); err != nil {
			return requestID, &ProviderError{Category: CategoryOutage, Code: "invalid_response", Message: "Seedance returned an invalid response", Retryable: method != http.MethodPost, Cause: err}
		}
	}
	return requestID, nil
}

func normalizeBytePlus(value bytePlusTask, requestID string) Task {
	status := strings.ToLower(strings.TrimSpace(value.Status))
	errorCode, errorMessage := "", ""
	if value.Error != nil {
		errorCode, errorMessage = value.Error.Code, safeProviderMessage(value.Error.Code)
	}
	usage := value.Usage.TotalTokens
	if usage == 0 {
		usage = value.Usage.CompletionTokens
	}
	result := Task{ID: value.ID, Model: value.Model, Status: status, OutputURL: value.Content.VideoURL, ErrorCode: errorCode, ErrorMessage: errorMessage, UsageTokens: usage, Seed: value.Seed, FPS: value.FramesPerSecond, Resolution: value.Resolution, AspectRatio: value.Ratio, DurationSeconds: value.Duration, GenerateAudio: value.GenerateAudio, ProviderRequestID: requestID}
	result.CreatedAt, result.UpdatedAt = unixTime(value.CreatedAt), unixTime(value.UpdatedAt)
	result.SafeResponse = map[string]any{"id": value.ID, "model": value.Model, "status": status, "outputAvailable": value.Content.VideoURL != "", "errorCode": errorCode, "usageTokens": usage, "resolution": value.Resolution, "ratio": value.Ratio, "duration": value.Duration, "generateAudio": value.GenerateAudio, "providerRequestId": requestID}
	return result
}

// ParseBytePlusTask decodes the same task representation returned by the
// retrieval endpoint and delivered to callback_url. Raw provider messages and
// temporary output URLs are deliberately excluded from SafeResponse.
func ParseBytePlusTask(payload []byte, requestID string) (Task, error) {
	var value bytePlusTask
	if err := json.Unmarshal(payload, &value); err != nil {
		return Task{}, &ProviderError{Category: CategoryInvalid, Code: "invalid_webhook", Message: "Seedance callback payload is invalid", Cause: err}
	}
	if value.ID == "" || value.Status == "" {
		return Task{}, &ProviderError{Category: CategoryInvalid, Code: "invalid_webhook", Message: "Seedance callback payload is incomplete"}
	}
	return normalizeBytePlus(value, requestID), nil
}

func validateCreate(input CreateRequest) error {
	if strings.TrimSpace(input.Prompt) == "" || strings.TrimSpace(input.Model) == "" || input.DurationSeconds < 2 || input.DurationSeconds > 15 || input.TimeoutSeconds <= 0 {
		return &ProviderError{Category: CategoryInvalid, Code: "invalid_request", Message: "Seedance prompt, model, duration, and timeout are required"}
	}
	if !map[string]bool{"480p": true, "720p": true, "1080p": true, "4k": true}[input.Resolution] || !map[string]bool{"16:9": true, "4:3": true, "1:1": true, "3:4": true, "9:16": true, "21:9": true, "adaptive": true}[input.AspectRatio] {
		return &ProviderError{Category: CategoryInvalid, Code: "format", Message: "Unsupported Seedance resolution or aspect ratio"}
	}
	images, videos, audios := 0, 0, 0
	for _, reference := range input.References {
		if strings.TrimSpace(reference.URL) == "" {
			return &ProviderError{Category: CategoryInvalid, Code: "reference_url", Message: "Seedance reference URL is required"}
		}
		switch reference.Type {
		case "image_url":
			images++
		case "video_url":
			videos++
		case "audio_url":
			audios++
		default:
			return &ProviderError{Category: CategoryInvalid, Code: "reference_type", Message: "Unsupported Seedance reference type"}
		}
	}
	if images > 9 || videos > 3 || audios > 3 || (audios > 0 && images+videos == 0) {
		return &ProviderError{Category: CategoryInvalid, Code: "reference_limits", Message: "Seedance reference limits were exceeded"}
	}
	return nil
}

func classifyHTTPError(status int, code string) error {
	lower := strings.ToLower(code)
	category, retryable, message := CategoryInvalid, false, "Seedance rejected the request"
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		category, message = CategoryAuthentication, "Seedance authentication failed"
	case status == http.StatusNotFound:
		category, message = CategoryNotFound, "Seedance task was not found"
	case status == http.StatusTooManyRequests || strings.Contains(lower, "burst") || strings.Contains(lower, "ratelimit"):
		category, retryable, message = CategoryRateLimit, true, "Seedance rate limit was reached"
	case strings.Contains(lower, "moderation") || strings.Contains(lower, "sensitive") || strings.Contains(lower, "safety"):
		category, message = CategoryModeration, "Seedance safety review rejected the request"
	case status >= 500:
		category, retryable, message = CategoryOutage, true, "Seedance is temporarily unavailable"
	}
	if code == "" {
		code = fmt.Sprintf("http_%d", status)
	}
	return &ProviderError{Category: category, Code: code, Message: message, Retryable: retryable}
}

func safeProviderMessage(code string) string {
	if strings.Contains(strings.ToLower(code), "moderation") || strings.Contains(strings.ToLower(code), "safety") {
		return "Seedance safety review rejected the task"
	}
	if code != "" {
		return "Seedance task failed"
	}
	return ""
}
func firstHeader(header http.Header, names ...string) string {
	for _, name := range names {
		if value := header.Get(name); value != "" {
			return value
		}
	}
	return ""
}

func unixTime(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	parsed := time.Unix(value, 0).UTC()
	return &parsed
}
