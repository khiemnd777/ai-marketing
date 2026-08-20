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

func rendererConfigured(cfg config.RendererConfig) bool {
	parsed, err := url.Parse(cfg.BaseURL)
	return err == nil && parsed.Scheme != "" && parsed.Host != "" && len(cfg.SharedSecret) >= 32
}
