package brands

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
	"strconv"
	"strings"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }
func ids(c fiber.Ctx, withBrand bool) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	clientID, e := uuid.Parse(c.Params("clientId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	workspaceID, e := uuid.Parse(c.Params("workspaceId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	brandID := uuid.Nil
	if withBrand {
		brandID, e = uuid.Parse(c.Params("brandId"))
		if e != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return clientID, workspaceID, brandID, nil
}
func (h *Handler) List(c fiber.Ctx) error {
	a, b, _, e := ids(c, false)
	if e != nil {
		return out(c, e)
	}
	v, e := h.service.List(c.Context(), a, b)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(fiber.Map{"items": v})
}
func (h *Handler) Get(c fiber.Ctx) error {
	a, b, id, e := ids(c, true)
	if e != nil {
		return out(c, e)
	}
	v, e := h.service.Get(c.Context(), a, b, id)
	if e != nil {
		return out(c, e)
	}
	c.Set("ETag", `W/"`+strconv.FormatInt(v.Version, 10)+`"`)
	return c.JSON(v)
}
func (h *Handler) Create(c fiber.Ctx) error {
	if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc.")
	}
	a, b, _, e := ids(c, false)
	if e != nil {
		return out(c, e)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.Create(c.Context(), a, b, actor.UserID, input)
	if e != nil {
		return out(c, e)
	}
	return c.Status(201).JSON(v)
}
func (h *Handler) Update(c fiber.Ctx) error {
	a, b, id, e := ids(c, true)
	if e != nil {
		return out(c, e)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.Update(c.Context(), a, b, id, actor.UserID, input)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(v)
}
func (h *Handler) SetStatus(c fiber.Ctx) error {
	a, b, id, e := ids(c, true)
	if e != nil {
		return out(c, e)
	}
	var input struct {
		Status  string `json:"status"`
		Version int64  `json:"version"`
	}
	if c.Bind().Body(&input) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.SetStatus(c.Context(), a, b, id, input.Status, input.Version, actor.UserID)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(v)
}
func out(c fiber.Ctx, e error) error {
	switch {
	case errors.Is(e, ErrInvalid):
		return problem.Write(c, 422, "validation", "Dữ liệu không hợp lệ", "Vui lòng kiểm tra hồ sơ thương hiệu.")
	case errors.Is(e, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy thương hiệu", "Thương hiệu không tồn tại trong workspace.")
	case errors.Is(e, ErrConflict):
		return problem.Write(c, 409, "version-conflict", "Dữ liệu đã thay đổi", "Tải lại hồ sơ trước khi lưu.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý thương hiệu", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
