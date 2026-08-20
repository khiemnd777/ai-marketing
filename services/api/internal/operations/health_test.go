package operations

import (
	"testing"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

func TestRendererConfigured(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.RendererConfig
		want bool
	}{
		{name: "valid internal endpoint", cfg: config.RendererConfig{BaseURL: "http://renderer:8090", SharedSecret: "renderer-secret-at-least-32-bytes"}, want: true},
		{name: "relative endpoint", cfg: config.RendererConfig{BaseURL: "renderer:8090", SharedSecret: "renderer-secret-at-least-32-bytes"}},
		{name: "short secret", cfg: config.RendererConfig{BaseURL: "https://renderer.example.com", SharedSecret: "too-short"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rendererConfigured(tt.cfg); got != tt.want {
				t.Fatalf("rendererConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}
