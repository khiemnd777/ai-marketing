package analytics

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func scope(c fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
	a, e := uuid.Parse(c.Params("clientId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, ErrInvalid
	}
	b, e := uuid.Parse(c.Params("workspaceId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, ErrInvalid
	}
	return a, b, nil
}
func filter(c fiber.Ctx) (Filter, error) {
	to := time.Now().UTC().AddDate(0, 0, 1).Truncate(24 * time.Hour)
	from := to.AddDate(0, 0, -30)
	var err error
	if raw := c.Query("from"); raw != "" {
		from, err = time.Parse("2006-01-02", raw)
		if err != nil {
			return Filter{}, ErrInvalid
		}
	}
	if raw := c.Query("to"); raw != "" {
		to, err = time.Parse("2006-01-02", raw)
		if err != nil {
			return Filter{}, ErrInvalid
		}
		to = to.AddDate(0, 0, 1)
	}
	var campaignID *uuid.UUID
	if raw := c.Query("campaignId"); raw != "" {
		id, e := uuid.Parse(raw)
		if e != nil {
			return Filter{}, ErrInvalid
		}
		campaignID = &id
	}
	return Filter{From: from, To: to, CampaignID: campaignID}, nil
}
func actor(c fiber.Ctx) uuid.UUID { p, _ := auth.PrincipalFrom(c); return p.UserID }
func (h *Handler) Summary(c fiber.Ctx) error {
	a, b, e := scope(c)
	if e != nil {
		return write(c, e)
	}
	f, e := filter(c)
	if e != nil {
		return write(c, e)
	}
	x, e := h.service.Summary(c.Context(), a, b, f)
	if e != nil {
		return write(c, e)
	}
	return c.JSON(x)
}
func (h *Handler) ListRecommendations(c fiber.Ctx) error {
	a, b, e := scope(c)
	if e != nil {
		return write(c, e)
	}
	f, e := filter(c)
	if e != nil {
		return write(c, e)
	}
	x, e := h.service.ListRecommendations(c.Context(), a, b, f.CampaignID)
	if e != nil {
		return write(c, e)
	}
	return c.JSON(fiber.Map{"items": x})
}
func (h *Handler) GenerateRecommendations(c fiber.Ctx) error {
	a, b, e := scope(c)
	if e != nil {
		return write(c, e)
	}
	f, e := filter(c)
	if e != nil {
		return write(c, e)
	}
	x, e := h.service.GenerateRecommendations(c.Context(), a, b, actor(c), f.CampaignID)
	if e != nil {
		return write(c, e)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"items": x})
}
func (h *Handler) ReviewRecommendation(c fiber.Ctx) error {
	a, b, e := scope(c)
	if e != nil {
		return write(c, e)
	}
	id, e := uuid.Parse(c.Params("recommendationId"))
	if e != nil {
		return write(c, ErrInvalid)
	}
	var input ReviewInput
	if c.Bind().Body(&input) != nil {
		return write(c, ErrInvalid)
	}
	x, e := h.service.ReviewRecommendation(c.Context(), a, b, id, actor(c), input)
	if e != nil {
		return write(c, e)
	}
	return c.JSON(x)
}
func write(c fiber.Ctx, e error) error {
	switch {
	case errors.Is(e, ErrInvalid):
		return problem.Write(c, 422, "analytics-invalid", "Bộ lọc analytics không hợp lệ", "Khoảng ngày tối đa 366 ngày và ID phải thuộc workspace.")
	case errors.Is(e, ErrNotFound):
		return problem.Write(c, 404, "analytics-not-found", "Không tìm thấy dữ liệu analytics", "Resource không thuộc workspace.")
	case errors.Is(e, ErrConflict):
		return problem.Write(c, 409, "analytics-conflict", "Recommendation đã thay đổi", "Tải lại trước khi review.")
	default:
		return problem.Write(c, 500, "analytics-error", "Không thể tải analytics", "Hệ thống chưa thể tổng hợp dữ liệu.")
	}
}
