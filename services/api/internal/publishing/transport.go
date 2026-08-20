package publishing

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
func scope(c fiber.Ctx, withPost bool) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, error) {
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
	postID := uuid.Nil
	if withPost {
		postID, err = uuid.Parse(c.Params("postId"))
		if err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return clientID, workspaceID, campaignID, postID, nil
}
func actor(c fiber.Ctx) uuid.UUID { principal, _ := auth.PrincipalFrom(c); return principal.UserID }
func (h *Handler) Create(c fiber.Ctx) error {
	a, b, p, _, err := scope(c, false)
	if err != nil {
		return writeError(c, err)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return writeError(c, ErrInvalid)
	}
	item, err := h.service.Create(c.Context(), a, b, p, actor(c), strings.TrimSpace(c.Get("Idempotency-Key")), input)
	if err != nil {
		return writeError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}
func (h *Handler) List(c fiber.Ctx) error {
	a, b, p, _, err := scope(c, false)
	if err != nil {
		return writeError(c, err)
	}
	items, err := h.service.List(c.Context(), a, b, p)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) Update(c fiber.Ctx) error {
	a, b, p, id, e := scope(c, true)
	if e != nil {
		return writeError(c, e)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return writeError(c, ErrInvalid)
	}
	principal, _ := auth.PrincipalFrom(c)
	item, e := h.service.Update(c.Context(), a, b, p, id, principal, auth.MetadataFrom(c), input)
	if e != nil {
		return writeError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) Review(c fiber.Ctx) error {
	a, b, p, id, err := scope(c, true)
	if err != nil {
		return writeError(c, err)
	}
	var input ReviewInput
	if c.Bind().Body(&input) != nil {
		return writeError(c, ErrInvalid)
	}
	item, err := h.service.Review(c.Context(), a, b, p, id, actor(c), input)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(item)
}
func writeError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalid):
		return problem.Write(c, 422, "social-post-invalid", "Bài đăng không hợp lệ", "Kiểm tra account, caption, media, lịch và version.")
	case errors.Is(err, ErrNotFound):
		return problem.Write(c, 404, "social-post-not-found", "Không tìm thấy bài đăng", "Bài đăng hoặc account không thuộc campaign này.")
	case errors.Is(err, ErrConflict):
		return problem.Write(c, 409, "social-post-conflict", "Bài đăng đã thay đổi", "Tải lại trước khi review.")
	case errors.Is(err, ErrPrerequisite):
		return problem.Write(c, 409, "publishing-prerequisite", "Chưa thể xuất bản", "Campaign cần final video đã duyệt và account Meta còn hiệu lực.")
	default:
		return problem.Write(c, 500, "social-post-error", "Không thể xử lý bài đăng", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
