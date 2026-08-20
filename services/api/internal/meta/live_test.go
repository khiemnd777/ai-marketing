package meta

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

func TestCreateCampaignAlwaysRequestsPaused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v99.0/act_123/campaigns" || request.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", request.Method, request.URL.Path)
		}
		if err := request.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if request.Form.Get("status") != "PAUSED" || request.Form.Get("special_ad_categories") != "[]" {
			t.Fatalf("unsafe campaign create form: %v", request.Form)
		}
		if request.Form.Get("access_token") != "page-token" || request.Form.Get("appsecret_proof") == "" {
			t.Fatal("authenticated server-side Graph fields are missing")
		}
		_ = json.NewEncoder(response).Encode(map[string]string{"id": "campaign-1"})
	}))
	defer server.Close()
	provider, err := NewLiveProvider(config.MetaConfig{AppID: "app", AppSecret: "secret", APIVersion: "v99.0", RedirectURL: "https://studio.example/meta/callback", GraphBaseURL: server.URL, DialogBaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.CreatePausedCampaign(context.Background(), CampaignRequest{AdAccountID: "123", Name: "Launch", Objective: "OUTCOME_TRAFFIC", BuyingType: "AUCTION", AccessToken: "page-token"})
	if err != nil || result.ProviderCampaignID != "campaign-1" {
		t.Fatalf("result=%#v error=%v", result, err)
	}
}

func TestMetaPermissionFailureIsNormalized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusForbidden)
		_, _ = response.Write([]byte(`{"error":{"code":200,"message":"raw confidential permission detail"}}`))
	}))
	defer server.Close()
	provider, err := NewLiveProvider(config.MetaConfig{AppID: "app", AppSecret: "secret", APIVersion: "v99.0", RedirectURL: "https://studio.example/meta/callback", GraphBaseURL: server.URL, DialogBaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = provider.CreatePausedCampaign(context.Background(), CampaignRequest{AdAccountID: "123", Name: "Launch", Objective: "OUTCOME_TRAFFIC", BuyingType: "AUCTION", AccessToken: "page-token"})
	var providerError *ProviderError
	if !errors.As(err, &providerError) || providerError.Category != "PERMISSION" || providerError.Retryable || providerError.Code != "200" {
		t.Fatalf("unexpected Meta error: %#v / %v", providerError, err)
	}
}

func TestInstagramPublishingUsesContainerThenPublish(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		_ = request.ParseForm()
		switch request.URL.Path {
		case "/v99.0/ig-1/media":
			if request.Form.Get("media_type") != "REELS" || request.Form.Get("video_url") == "" {
				t.Fatalf("invalid container form: %v", request.Form)
			}
			_ = json.NewEncoder(response).Encode(map[string]string{"id": "container-1"})
		case "/v99.0/container-1":
			_ = json.NewEncoder(response).Encode(map[string]string{"status_code": "FINISHED"})
		case "/v99.0/ig-1/media_publish":
			if request.Form.Get("creation_id") != "container-1" {
				t.Fatalf("invalid publish form: %v", request.Form)
			}
			_ = json.NewEncoder(response).Encode(map[string]string{"id": "post-1"})
		case "/v99.0/post-1":
			_ = json.NewEncoder(response).Encode(map[string]string{"permalink": "https://instagram.example/p/post-1"})
		default:
			t.Fatalf("unexpected request path %s", request.URL.Path)
		}
	}))
	defer server.Close()
	provider, _ := NewLiveProvider(config.MetaConfig{AppID: "app", AppSecret: "secret", APIVersion: "v99.0", RedirectURL: "https://studio.example/meta/callback", GraphBaseURL: server.URL, DialogBaseURL: server.URL})
	result, err := provider.Publish(context.Background(), PublishRequest{Platform: "INSTAGRAM", InstagramID: "ig-1", MediaURL: "https://signed.example/final.mp4", MediaType: "video/mp4", Caption: "Xin chào", AccessToken: "page-token"})
	if err != nil || result.ProviderPostID != "post-1" || result.PublicURL == "" || requests != 4 {
		t.Fatalf("result=%#v requests=%d error=%v", result, requests, err)
	}
}
