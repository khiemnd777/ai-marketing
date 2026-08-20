package operations

import (
	"context"
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
			{"name": "seedance", "configured": h.config.Seedance.APIKey != "" && h.config.Seedance.BaseURL != "" && h.config.Seedance.Model != "", "model": h.config.Seedance.Model, "apiVersion": h.config.Seedance.APIVersion, "baseUrl": safeHost(h.config.Seedance.BaseURL)},
			{"name": "r2", "configured": h.config.R2.Validate() == nil, "bucket": h.config.R2.Bucket, "baseUrl": safeHost(h.config.R2.Endpoint)},
			{"name": "meta", "configured": h.config.Meta.AppID != "" && h.config.Meta.AppSecret != "" && h.config.Meta.APIVersion != "", "apiVersion": h.config.Meta.APIVersion},
			{"name": "renderer", "configured": h.config.Renderer.BaseURL != "" && h.config.Renderer.SharedSecret != "", "baseUrl": safeHost(h.config.Renderer.BaseURL)},
		},
	})
}

func safeHost(value string) string {
	for index, character := range value {
		if character == '?' || character == '#' {
			return value[:index]
		}
	}
	return value
}
