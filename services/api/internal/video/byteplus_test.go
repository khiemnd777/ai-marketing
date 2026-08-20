package video

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

func TestBytePlusProviderCreateGetAndCancel(t *testing.T) {
	t.Parallel()
	var methods []string
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		methods = append(methods, r.Method)
		if got := r.Header.Get("Authorization"); got != "Bearer secret-value" {
			t.Fatalf("unexpected authorization header %q", got)
		}
		response := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"x-request-id": []string{"provider-request"}}, Body: io.NopCloser(strings.NewReader(""))}
		switch r.Method {
		case http.MethodPost:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["model"] != "dreamina-seedance-2-0-260128" || body["ratio"] != "9:16" || body["resolution"] != "720p" || body["callback_url"] != "https://studio.test/webhooks/seedance?token=opaque" {
				t.Fatalf("unexpected create body %#v", body)
			}
			content, ok := body["content"].([]any)
			if !ok || len(content) != 2 {
				t.Fatalf("unexpected content %#v", body["content"])
			}
			response.StatusCode = http.StatusCreated
			response.Body = io.NopCloser(strings.NewReader(`{"id":"task-1","model":"dreamina-seedance-2-0-260128","status":"queued","created_at":1787190000}`))
		case http.MethodGet:
			response.Body = io.NopCloser(strings.NewReader(`{"id":"task-1","model":"dreamina-seedance-2-0-260128","status":"succeeded","content":{"video_url":"https://output.example/video.mp4"},"usage":{"total_tokens":4200},"resolution":"720p","ratio":"9:16","duration":5,"generate_audio":true}`))
		case http.MethodDelete:
			response.StatusCode = http.StatusNoContent
		}
		return response, nil
	})

	provider, err := NewBytePlusProvider(config.SeedanceConfig{APIKey: "secret-value", BaseURL: "https://ark.test/api", APIVersion: "v3", Model: "dreamina-seedance-2-0-260128", Timeout: time.Second, PollInterval: time.Second, TaskTimeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	provider.client = &http.Client{Transport: transport}
	input := CreateRequest{Prompt: "A precise scene", References: []Reference{{Type: "image_url", URL: "https://assets.test/product.jpg", Role: "reference_image"}}, Model: "dreamina-seedance-2-0-260128", Resolution: "720p", AspectRatio: "9:16", DurationSeconds: 5, GenerateAudio: true, CallbackURL: "https://studio.test/webhooks/seedance?token=opaque", TimeoutSeconds: 300}
	created, err := provider.Create(context.Background(), input)
	if err != nil || created.ID != "task-1" || created.Status != "queued" {
		t.Fatalf("create: task=%#v error=%v", created, err)
	}
	completed, err := provider.Get(context.Background(), created.ID)
	if err != nil || completed.Status != "succeeded" || completed.OutputURL == "" || completed.UsageTokens != 4200 {
		t.Fatalf("get: task=%#v error=%v", completed, err)
	}
	if err := provider.Cancel(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}
	if strings.Join(methods, ",") != "POST,GET,DELETE" {
		t.Fatalf("unexpected methods %v", methods)
	}
}

func TestBytePlusProviderSanitizesProviderErrors(t *testing.T) {
	t.Parallel()
	provider, err := NewBytePlusProvider(config.SeedanceConfig{APIKey: "secret", BaseURL: "https://ark.test/api", APIVersion: "v3", Model: "model", Timeout: time.Second, PollInterval: time.Second, TaskTimeout: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	provider.client = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"code":"SensitiveContentDetected","message":"raw confidential moderation explanation"}}`))}, nil
	})}
	_, err = provider.Create(context.Background(), CreateRequest{Prompt: "scene", Model: "model", Resolution: "720p", AspectRatio: "9:16", DurationSeconds: 5, TimeoutSeconds: 60})
	providerError := AsProviderError(err)
	if providerError.Category != CategoryModeration || strings.Contains(providerError.Error(), "confidential") {
		t.Fatalf("provider error was not normalized: %#v", providerError)
	}
}

func TestBytePlusProviderClassifiesTimeoutAndRateLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		transport roundTripFunc
		category  ErrorCategory
		retryable bool
	}{
		{"timeout", func(request *http.Request) (*http.Response, error) { return nil, request.Context().Err() }, CategoryTimeout, true},
		{"rate-limit", func(_ *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusTooManyRequests, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"code":"RateLimitExceeded"}}`))}, nil
		}, CategoryRateLimit, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider, err := NewBytePlusProvider(config.SeedanceConfig{APIKey: "secret", BaseURL: "https://ark.test/api", APIVersion: "v3", Model: "model", Timeout: time.Second, PollInterval: time.Second, TaskTimeout: time.Minute})
			if err != nil {
				t.Fatal(err)
			}
			provider.client = &http.Client{Transport: test.transport}
			ctx := context.Background()
			if test.name == "timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			_, err = provider.Get(ctx, "task-1")
			providerError := AsProviderError(err)
			if providerError.Category != test.category || providerError.Retryable != test.retryable {
				t.Fatalf("category=%s retryable=%t error=%v", providerError.Category, providerError.Retryable, err)
			}
		})
	}
}

func TestParseBytePlusTaskSanitizesTemporaryOutput(t *testing.T) {
	t.Parallel()
	task, err := ParseBytePlusTask([]byte(`{"id":"task-webhook","status":"succeeded","content":{"video_url":"https://temporary.volces.com/output.mp4"},"usage":{"total_tokens":900}}`), "callback-request")
	if err != nil {
		t.Fatal(err)
	}
	if task.OutputURL == "" || task.ProviderRequestID != "callback-request" {
		t.Fatalf("unexpected normalized callback %#v", task)
	}
	encoded, err := json.Marshal(task.SafeResponse)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "temporary.volces.com") || task.SafeResponse["outputAvailable"] != true {
		t.Fatalf("safe callback response leaked temporary URL: %s", encoded)
	}
}

func TestProviderStateOnlyMovesForward(t *testing.T) {
	t.Parallel()
	if !mayAdvance("PROVIDER_QUEUED", "PROVIDER_PROCESSING") || !mayAdvance("PROVIDER_PROCESSING", "SUCCEEDED") {
		t.Fatal("expected forward provider transitions")
	}
	if mayAdvance("SUCCEEDED", "PROVIDER_PROCESSING") || mayAdvance("APPROVED", "FAILED") {
		t.Fatal("terminal or regressive transition was accepted")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
