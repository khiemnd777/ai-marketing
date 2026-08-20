package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

func TestDemoProviderOutputsPassDeterministicValidation(t *testing.T) {
	provider := NewDemoProvider("demo")
	ctx := PlanningContext{CampaignID: uuid.New(), CampaignName: "Launch", Objective: "AWARENESS", Audience: "Travelers", Market: "Vietnam", Language: "vi", VideoFormat: "INTERVIEW_REVIEW", DurationSeconds: 30, Tone: "Practical", CTA: "Tìm hiểu thêm", BrandName: "Northstar", ProductName: "Cabin 20", ProductTruth: []ProductFact{{ID: uuid.New(), Key: "external_dimensions", ExactValue: "55 x 36 x 23 cm", Locked: true}}}
	concepts, _, err := provider.GenerateConcepts(context.Background(), ConceptInput{Context: ctx})
	if err != nil || ValidateConcepts(concepts) != nil {
		t.Fatalf("concepts invalid: %v / %v", err, ValidateConcepts(concepts))
	}
	content, _, err := provider.GenerateContent(context.Background(), ContentInput{Context: ctx})
	if err != nil || ValidateContent(content, ctx) != nil {
		t.Fatalf("content invalid: %v / %v", err, ValidateContent(content, ctx))
	}
	script, _, err := provider.GenerateScript(context.Background(), ScriptInput{Context: ctx})
	if err != nil || ValidateScript(script, ctx) != nil {
		t.Fatalf("script invalid: %v / %v", err, ValidateScript(script, ctx))
	}
	sceneInput := SceneInput{Context: ctx, Script: script, SpeakerCharacterID: uuid.New(), ListenerCharacterID: uuid.New()}
	scenes, _, err := provider.GenerateScenes(context.Background(), sceneInput)
	if err != nil || ValidateScenes(scenes, sceneInput) != nil {
		t.Fatalf("scenes invalid: %v / %v", err, ValidateScenes(scenes, sceneInput))
	}
}

func TestOpenAIProviderRejectsSchemaFailureAndRedactsProviderMessage(t *testing.T) {
	t.Parallel()
	provider, err := NewOpenAIProvider(config.OpenAIConfig{APIKey: "secret", BaseURL: "https://openai.test/v1", Model: "fixture", ReasoningEffort: "medium", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	provider.client = &http.Client{Transport: aiRoundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"x-request-id": []string{"request-fixture"}}, Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","code":"schema_validation_failed","message":"secret raw provider detail"}}`))}, nil
	})}
	_, _, err = provider.GenerateConcepts(context.Background(), ConceptInput{})
	if err == nil || strings.Contains(err.Error(), "secret raw provider detail") || !strings.Contains(err.Error(), "schema_validation_failed") {
		t.Fatalf("provider error was not safely normalized: %v", err)
	}
}

func TestOpenAIProviderFailureMatrix(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		status    int
		body      string
		category  string
		retryable bool
	}{
		{"authentication", http.StatusUnauthorized, `{"error":{"type":"authentication_error","code":"invalid_api_key","message":"raw key detail"}}`, "AUTHENTICATION", false},
		{"rate-limit", http.StatusTooManyRequests, `{"error":{"type":"rate_limit_error","code":"rate_limit_exceeded","message":"raw quota detail"}}`, "RATE_LIMIT", true},
		{"outage", http.StatusServiceUnavailable, `{"error":{"type":"server_error","code":"overloaded","message":"raw incident detail"}}`, "OUTAGE", true},
		{"moderation", http.StatusBadRequest, `{"error":{"type":"invalid_request_error","code":"content_policy_violation","message":"raw moderation detail"}}`, "MODERATION", false},
		{"malformed-envelope", http.StatusOK, `{not-json`, "PROTOCOL", false},
		{"invalid-structured-output", http.StatusOK, `{"id":"response-1","output":[{"content":[{"type":"output_text","text":"{\"concepts\":\"wrong\"}"}]}]}`, "VALIDATION", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider, err := NewOpenAIProvider(config.OpenAIConfig{APIKey: "secret", BaseURL: "https://openai.test/v1", Model: "fixture", ReasoningEffort: "medium", Timeout: time.Second})
			if err != nil {
				t.Fatal(err)
			}
			provider.client = &http.Client{Transport: aiRoundTripFunc(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: test.status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})}
			_, _, err = provider.GenerateConcepts(context.Background(), ConceptInput{})
			providerError := AsProviderError(err)
			if providerError.Category != test.category || providerError.Retryable != test.retryable {
				t.Fatalf("category=%s retryable=%t err=%v", providerError.Category, providerError.Retryable, err)
			}
			if strings.Contains(err.Error(), "raw ") {
				t.Fatalf("provider detail leaked: %v", err)
			}
		})
	}
}

func TestOpenAIProviderClassifiesTimeout(t *testing.T) {
	t.Parallel()
	provider, err := NewOpenAIProvider(config.OpenAIConfig{APIKey: "secret", BaseURL: "https://openai.test/v1", Model: "fixture", ReasoningEffort: "medium", Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	provider.client = &http.Client{Transport: aiRoundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})}
	_, _, err = provider.GenerateConcepts(context.Background(), ConceptInput{})
	providerError := AsProviderError(err)
	if !errors.Is(err, context.DeadlineExceeded) || providerError.Category != "TIMEOUT" || !providerError.Retryable {
		t.Fatalf("unexpected timeout classification: %#v / %v", providerError, err)
	}
}

type aiRoundTripFunc func(*http.Request) (*http.Response, error)

func (function aiRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
