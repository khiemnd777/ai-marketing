package operations

import (
	"context"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type Handler struct {
	pool   *pgxpool.Pool
	config config.Config
}

func NewHandler(pool *pgxpool.Pool, cfg config.Config) *Handler {
	return &Handler{pool: pool, config: cfg}
}

func (h *Handler) Liveness(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok", "timestamp": time.Now().UTC()})
}

func (h *Handler) Readiness(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		return problem.Write(c, fiber.StatusServiceUnavailable, "not-ready", "Dịch vụ chưa sẵn sàng", "Kết nối cơ sở dữ liệu chưa sẵn sàng.")
	}
	return c.JSON(fiber.Map{"status": "ok", "timestamp": time.Now().UTC(), "checks": fiber.Map{"database": "ok"}})
}

func (h *Handler) ProviderStatus(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"demoMode": h.config.DemoMode,
		"providers": []fiber.Map{
			{"name": "openai", "configured": h.config.OpenAI.Validate() == nil, "model": h.config.OpenAI.Model, "baseUrl": safeHost(h.config.OpenAI.BaseURL)},
			{"name": "seedance", "configured": h.config.Seedance.Validate() == nil, "model": h.config.Seedance.Model, "apiVersion": h.config.Seedance.APIVersion, "baseUrl": safeHost(h.config.Seedance.BaseURL)},
			{"name": "r2", "configured": h.config.R2.Validate() == nil, "bucket": h.config.R2.Bucket, "baseUrl": safeHost(h.config.R2.Endpoint)},
			{"name": "meta", "configured": h.config.Meta.Validate() == nil, "apiVersion": h.config.Meta.APIVersion, "baseUrl": safeHost(h.config.Meta.GraphBaseURL)},
			{"name": "renderer", "configured": rendererConfigured(h.config.Renderer), "baseUrl": safeHost(h.config.Renderer.BaseURL)},
		},
	})
}

func rendererConfigured(cfg config.RendererConfig) bool {
	parsed, err := url.Parse(cfg.BaseURL)
	return err == nil && parsed.Scheme != "" && parsed.Host != "" && len(cfg.SharedSecret) >= 32
}

func safeHost(value string) string {
	for index, character := range value {
		if character == '?' || character == '#' {
			return value[:index]
		}
	}
	return value
}
