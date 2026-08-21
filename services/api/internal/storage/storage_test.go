package storage

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

func TestS3StoreUsesBrowserEndpointOnlyForPresignedRequests(t *testing.T) {
	t.Parallel()

	store, err := NewS3Store(context.Background(), config.R2Config{
		AccessKeyID:     "test-access",
		SecretAccessKey: "test-secret",
		Bucket:          "media",
		Endpoint:        "http://minio:9000",
		BrowserEndpoint: "http://localhost:9100",
	})
	if err != nil {
		t.Fatalf("NewS3Store() error = %v", err)
	}
	presigned, err := store.PresignPut(context.Background(), "workspaces/test/assets/test/original.png", "image/png", 123, 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignPut() error = %v", err)
	}
	presignedURL, err := url.Parse(presigned.URL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	if presignedURL.Host != "localhost:9100" {
		t.Fatalf("presigned host = %q, want browser host", presignedURL.Host)
	}
	clientOptions := store.client.Options()
	if clientOptions.BaseEndpoint == nil || *clientOptions.BaseEndpoint != "http://minio:9000" {
		t.Fatalf("internal client endpoint = %v, want http://minio:9000", clientOptions.BaseEndpoint)
	}
}

func TestPresignedRequestJSONMatchesOpenAPIContract(t *testing.T) {
	t.Parallel()
	request := PresignedRequest{
		URL:       "https://storage.example/upload",
		Method:    "PUT",
		Headers:   map[string]string{},
		ExpiresAt: time.Date(2026, time.August, 21, 3, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var body map[string]any
	if err = json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	for _, key := range []string{"url", "method", "headers", "expiresAt"} {
		if _, ok := body[key]; !ok {
			t.Errorf("JSON body is missing contract key %q: %s", key, raw)
		}
	}
	for _, key := range []string{"URL", "Method", "Headers", "ExpiresAt"} {
		if _, ok := body[key]; ok {
			t.Errorf("JSON body exposes Go field name %q: %s", key, raw)
		}
	}
}

func TestValidateObjectKey(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{name: "scoped", key: "workspaces/a/assets/b/original.mp4", valid: true},
		{name: "parent traversal", key: "workspaces/a/../secret", valid: false},
		{name: "absolute", key: "/workspaces/a", valid: false},
		{name: "backslash", key: "workspaces\\a", valid: false},
		{name: "empty segment", key: "workspaces//a", valid: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateObjectKey(test.key)
			if test.valid && err != nil {
				t.Fatalf("validateObjectKey() error = %v", err)
			}
			if !test.valid && !errors.Is(err, ErrInvalidObjectKey) {
				t.Fatalf("validateObjectKey() error = %v, want ErrInvalidObjectKey", err)
			}
		})
	}
}

func TestScopedObjectKeyDoesNotKeepFilename(t *testing.T) {
	key, err := ScopedObjectKey(uuid.New(), uuid.New(), "customer passport scan.JPG")
	if err != nil {
		t.Fatalf("ScopedObjectKey() error = %v", err)
	}
	if strings.Contains(key, "passport") || !strings.HasSuffix(key, ".jpg") {
		t.Fatalf("ScopedObjectKey() = %q", key)
	}
}
