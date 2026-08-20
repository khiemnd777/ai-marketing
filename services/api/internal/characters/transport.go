package characters

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func characterScope(c fiber.Ctx, campaign bool) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	clientID, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	workspaceID, err := uuid.Parse(c.Params("workspaceId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	campaignID := uuid.Nil
	if campaign {
		campaignID, err = uuid.Parse(c.Params("campaignId"))
		if err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return clientID, workspaceID, campaignID, nil
}
func (h *Handler) List(c fiber.Ctx) error {
	a, b, _, e := characterScope(c, false)
	if e != nil {
		return characterError(c, e)
	}
	items, e := h.service.List(c.Context(), a, b)
	if e != nil {
		return characterError(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) Create(c fiber.Ctx) error {
	if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc.")
	}
	a, b, _, e := characterScope(c, false)
	if e != nil {
		return characterError(c, e)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return characterError(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	item, e := h.service.Create(c.Context(), a, b, actor.UserID, input)
	if e != nil {
		return characterError(c, e)
	}
	return c.Status(201).JSON(item)
}
func (h *Handler) GetSelection(c fiber.Ctx) error {
	a, b, id, e := characterScope(c, true)
	if e != nil {
		return characterError(c, e)
	}
	item, e := h.service.GetSelection(c.Context(), a, b, id)
	if e != nil {
		return characterError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) Select(c fiber.Ctx) error {
	a, b, id, e := characterScope(c, true)
	if e != nil {
		return characterError(c, e)
	}
	var input struct {
		PrimaryCharacterID  uuid.UUID `json:"primaryCharacterId"`
		ListenerCharacterID uuid.UUID `json:"listenerCharacterId"`
	}
	if c.Bind().Body(&input) != nil {
		return characterError(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	item, e := h.service.Select(c.Context(), a, b, id, input.PrimaryCharacterID, input.ListenerCharacterID, actor.UserID)
	if e != nil {
		return characterError(c, e)
	}
	return c.JSON(item)
}
func characterError(c fiber.Ctx, e error) error {
	switch {
	case errors.Is(e, ErrInvalid):
		return problem.Write(c, 422, "validation", "Nhân vật không hợp lệ", "Chọn đúng hai nhân vật khác nhau, đang hoạt động và có trạng thái đồng ý phù hợp.")
	case errors.Is(e, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy nhân vật", "Nhân vật hoặc campaign không thuộc workspace.")
	case errors.Is(e, ErrConflict):
		return problem.Write(c, 409, "conflict", "Nhân vật đã thay đổi", "Tải lại dữ liệu trước khi lưu.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý nhân vật", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
