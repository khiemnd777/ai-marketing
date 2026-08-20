package metaads

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
	"strings"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func ids(c fiber.Ctx, withCampaign, withAction bool) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, error) {
	clientID, e := uuid.Parse(c.Params("clientId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	workspaceID, e := uuid.Parse(c.Params("workspaceId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	campaignID := uuid.Nil
	if c.Params("campaignId") != "" {
		campaignID, e = uuid.Parse(c.Params("campaignId"))
		if e != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	adID := uuid.Nil
	if withCampaign {
		adID, e = uuid.Parse(c.Params("adCampaignId"))
		if e != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	actionID := uuid.Nil
	if withAction {
		actionID, e = uuid.Parse(c.Params("actionId"))
		if e != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return clientID, workspaceID, campaignID, adID, actionID, nil
}
func user(c fiber.Ctx) uuid.UUID { p, _ := auth.PrincipalFrom(c); return p.UserID }
func (h *Handler) GetGuardrails(c fiber.Ctx) error {
	a, b, _, _, _, e := ids(c, false, false)
	if e != nil {
		return adError(c, e)
	}
	item, e := h.service.GetGuardrails(c.Context(), a, b)
	if e != nil {
		return adError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) SaveGuardrails(c fiber.Ctx) error {
	a, b, _, _, _, e := ids(c, false, false)
	if e != nil {
		return adError(c, e)
	}
	var input Guardrails
	if c.Bind().Body(&input) != nil {
		return adError(c, ErrInvalid)
	}
	item, e := h.service.SaveGuardrails(c.Context(), a, b, user(c), input)
	if e != nil {
		return adError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) Create(c fiber.Ctx) error {
	a, b, p, _, _, e := ids(c, false, false)
	if e != nil {
		return adError(c, e)
	}
	var input CampaignInput
	if c.Bind().Body(&input) != nil {
		return adError(c, ErrInvalid)
	}
	item, e := h.service.Create(c.Context(), a, b, p, user(c), strings.TrimSpace(c.Get("Idempotency-Key")), input)
	if e != nil {
		return adError(c, e)
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}
func (h *Handler) List(c fiber.Ctx) error {
	a, b, p, _, _, e := ids(c, false, false)
	if e != nil {
		return adError(c, e)
	}
	items, e := h.service.List(c.Context(), a, b, p)
	if e != nil {
		return adError(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) ReviewCreate(c fiber.Ctx) error {
	a, b, p, id, _, e := ids(c, true, false)
	if e != nil {
		return adError(c, e)
	}
	var input ReviewInput
	if c.Bind().Body(&input) != nil {
		return adError(c, ErrInvalid)
	}
	item, e := h.service.ReviewCreate(c.Context(), a, b, p, id, user(c), input)
	if e != nil {
		return adError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) RequestAction(c fiber.Ctx) error {
	a, b, p, id, _, e := ids(c, true, false)
	if e != nil {
		return adError(c, e)
	}
	var input ActionInput
	if c.Bind().Body(&input) != nil {
		return adError(c, ErrInvalid)
	}
	input.IdempotencyKey = strings.TrimSpace(c.Get("Idempotency-Key"))
	item, e := h.service.RequestAction(c.Context(), a, b, p, id, user(c), input)
	if e != nil {
		return adError(c, e)
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}
func (h *Handler) ListActions(c fiber.Ctx) error {
	a, b, p, id, _, e := ids(c, true, false)
	if e != nil {
		return adError(c, e)
	}
	items, e := h.service.ListActions(c.Context(), a, b, p, id)
	if e != nil {
		return adError(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) ReviewAction(c fiber.Ctx) error {
	a, b, p, id, actionID, e := ids(c, true, true)
	if e != nil {
		return adError(c, e)
	}
	var input ReviewInput
	if c.Bind().Body(&input) != nil {
		return adError(c, ErrInvalid)
	}
	item, e := h.service.ReviewAction(c.Context(), a, b, p, id, actionID, user(c), input)
	if e != nil {
		return adError(c, e)
	}
	return c.JSON(item)
}
func adError(c fiber.Ctx, e error) error {
	switch {
	case errors.Is(e, ErrInvalid):
		return problem.Write(c, 422, "meta-ad-invalid", "Meta Ads không hợp lệ", "Kiểm tra objective, audience, budget, URL, creative và version.")
	case errors.Is(e, ErrNotFound):
		return problem.Write(c, 404, "meta-ad-not-found", "Không tìm thấy Meta Ads resource", "Resource không thuộc workspace hoặc campaign.")
	case errors.Is(e, ErrConflict):
		return problem.Write(c, 409, "meta-ad-conflict", "Meta Ads đã thay đổi", "Tải lại trước khi tiếp tục.")
	case errors.Is(e, ErrGuardrail):
		return problem.Write(c, 409, "meta-ad-guardrail", "Budget guardrail đã chặn thao tác", "Xác nhận đúng budget và giữ mức chi trong workspace/campaign cap.")
	case errors.Is(e, ErrPrerequisite):
		return problem.Write(c, 409, "meta-ad-prerequisite", "Chưa đủ điều kiện tạo Ads", "Cần Meta connection, final render, creative và guardrails hợp lệ.")
	default:
		return problem.Write(c, 500, "meta-ad-error", "Không thể xử lý Meta Ads", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
