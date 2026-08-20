package rendering

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
func renderScope(c fiber.Ctx, withJob bool) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, error) {
	clientID, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	workspaceID, err := uuid.Parse(c.Params("workspaceId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	campaignID, err := uuid.Parse(c.Params("campaignId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	jobID := uuid.Nil
	if withJob {
		jobID, err = uuid.Parse(c.Params("renderJobId"))
		if err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return clientID, workspaceID, campaignID, jobID, nil
}
func renderActor(c fiber.Ctx) uuid.UUID {
	principal, _ := auth.PrincipalFrom(c)
	return principal.UserID
}
func (h *Handler) GetProject(c fiber.Ctx) error {
	a, b, p, _, e := renderScope(c, false)
	if e != nil {
		return renderError(c, e)
	}
	item, e := h.service.GetProject(c.Context(), a, b, p)
	if e != nil {
		return renderError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) SaveProject(c fiber.Ctx) error {
	a, b, p, _, e := renderScope(c, false)
	if e != nil {
		return renderError(c, e)
	}
	var input ProjectInput
	if c.Bind().Body(&input) != nil {
		return renderError(c, ErrInvalid)
	}
	item, e := h.service.SaveProject(c.Context(), a, b, p, renderActor(c), input)
	if e != nil {
		return renderError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) Start(c fiber.Ctx) error {
	a, b, p, _, e := renderScope(c, false)
	if e != nil {
		return renderError(c, e)
	}
	key := strings.TrimSpace(c.Get("Idempotency-Key"))
	if key == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc cho final render.")
	}
	item, e := h.service.Start(c.Context(), a, b, p, renderActor(c), key)
	if e != nil {
		return renderError(c, e)
	}
	return c.Status(202).JSON(item)
}
func (h *Handler) List(c fiber.Ctx) error {
	a, b, p, _, e := renderScope(c, false)
	if e != nil {
		return renderError(c, e)
	}
	items, e := h.service.List(c.Context(), a, b, p)
	if e != nil {
		return renderError(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) Review(c fiber.Ctx) error {
	a, b, p, id, e := renderScope(c, true)
	if e != nil {
		return renderError(c, e)
	}
	var input ReviewInput
	if c.Bind().Body(&input) != nil {
		return renderError(c, ErrInvalid)
	}
	item, e := h.service.Review(c.Context(), a, b, p, id, renderActor(c), input)
	if e != nil {
		return renderError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) Select(c fiber.Ctx) error {
	a, b, p, id, e := renderScope(c, true)
	if e != nil {
		return renderError(c, e)
	}
	item, e := h.service.Select(c.Context(), a, b, p, id, renderActor(c))
	if e != nil {
		return renderError(c, e)
	}
	return c.JSON(item)
}
func renderError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalid):
		return problem.Write(c, 422, "validation", "Final composer không hợp lệ", "Kiểm tra headline, music, gain, phiên bản và review notes.")
	case errors.Is(err, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy final project", "Project hoặc render job không thuộc campaign này.")
	case errors.Is(err, ErrConflict):
		return problem.Write(c, 409, "render-conflict", "Final project đã thay đổi", "Tải lại project hoặc render job trước khi tiếp tục.")
	case errors.Is(err, ErrPrerequisite):
		return problem.Write(c, 409, "scene-selection-required", "Chưa đủ take đã duyệt", "Mỗi scene hiện tại phải có đúng một take đã duyệt và được chọn.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý final render", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
