package rendering

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

func TestRendererClientSignsExactManifest(t *testing.T) {
	const secret = "renderer-test-secret-at-least-32-bytes"
	raw := []byte(`{"renderId":"018f47a0-7b5f-7d5f-9d2a-c5939813086f"}`)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != string(raw) {
			t.Fatalf("body changed across renderer boundary: %q", body)
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(raw)
		if request.Header.Get("X-Render-Signature") != hex.EncodeToString(mac.Sum(nil)) {
			t.Fatal("renderer signature does not cover the exact manifest")
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"requestId":"request-1","reused":false,"outputObjectKey":"final.mp4","thumbnailObjectKey":"thumb.jpg","checksumSha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","fileSizeBytes":1024,"width":1080,"height":1920,"fps":30,"durationMs":30000,"codec":"h264","audioCodec":"aac"}`)
	}))
	defer server.Close()

	client, err := NewRendererClient(config.RendererConfig{BaseURL: server.URL, SharedSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Render(context.Background(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequestID != "request-1" || result.Width != 1080 || result.Height != 1920 || result.Codec != "h264" {
		t.Fatalf("unexpected normalized result: %#v", result)
	}
}

func TestRendererClientSignsClientStorageEnvelope(t *testing.T) {
	const secret = "renderer-test-secret-at-least-32-bytes"
	manifest := []byte(`{"renderId":"018f47a0-7b5f-7d5f-9d2a-c5939813086f"}`)
	storageConfig := config.R2Config{Endpoint: "https://tenant-storage.example.test", AccessKeyID: "tenant-access", SecretAccessKey: "tenant-secret", Bucket: "tenant-bucket"}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		if request.Header.Get("X-Render-Signature") != hex.EncodeToString(mac.Sum(nil)) {
			t.Fatal("renderer signature does not cover the storage envelope")
		}
		var envelope struct {
			Manifest map[string]any    `json:"manifest"`
			Storage  map[string]string `json:"storage"`
		}
		if err = json.Unmarshal(body, &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Manifest["renderId"] == "" || envelope.Storage["accessKeyId"] != storageConfig.AccessKeyID || envelope.Storage["secretAccessKey"] != storageConfig.SecretAccessKey || envelope.Storage["bucket"] != storageConfig.Bucket {
			t.Fatalf("unexpected renderer envelope: %#v", envelope)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(response, `{"requestId":"request-2","reused":false,"outputObjectKey":"final.mp4","thumbnailObjectKey":"thumb.jpg","checksumSha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","fileSizeBytes":1024,"width":1080,"height":1920,"fps":30,"durationMs":30000,"codec":"h264","audioCodec":"aac"}`)
	}))
	defer server.Close()

	client, err := NewRendererClient(config.RendererConfig{BaseURL: server.URL, SharedSecret: secret})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.RenderWithStorage(context.Background(), manifest, storageConfig); err != nil {
		t.Fatal(err)
	}
}

func TestFinalRenderWorkerTimeoutOutlivesRendererRequest(t *testing.T) {
	client, err := NewRendererClient(config.RendererConfig{
		BaseURL:      "https://renderer.test",
		SharedSecret: "renderer-test-secret-at-least-32-bytes",
	})
	if err != nil {
		t.Fatal(err)
	}
	workerTimeout := (&Worker{}).Timeout(nil)
	if workerTimeout <= client.client.Timeout {
		t.Fatalf("worker timeout %s must outlive renderer request timeout %s", workerTimeout, client.client.Timeout)
	}
}

func TestSubtitlesPreserveVietnameseAndFormat(t *testing.T) {
	srt, vtt := subtitles([]CaptionCue{{StartMS: 1250, EndMS: 3650, Text: "Hành trình nhẹ nhàng"}})
	if !strings.Contains(string(srt), "00:00:01,250 --> 00:00:03,650\nHành trình nhẹ nhàng") {
		t.Fatalf("unexpected SRT: %s", srt)
	}
	if !strings.HasPrefix(string(vtt), "WEBVTT\n\n00:00:01.250 --> 00:00:03.650") || !strings.Contains(string(vtt), "Hành trình nhẹ nhàng") {
		t.Fatalf("unexpected VTT: %s", vtt)
	}
}
