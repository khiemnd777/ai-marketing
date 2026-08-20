package products

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
func scope(c fiber.Ctx, product bool) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	a, e := uuid.Parse(c.Params("clientId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	b, e := uuid.Parse(c.Params("workspaceId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	id := uuid.Nil
	if product {
		id, e = uuid.Parse(c.Params("productId"))
		if e != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return a, b, id, nil
}
func (h *Handler) List(c fiber.Ctx) error {
	a, b, _, e := scope(c, false)
	if e != nil {
		return out(c, e)
	}
	items, e := h.service.List(c.Context(), a, b, c.Query("search"), c.Query("status"))
	if e != nil {
		return out(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) Get(c fiber.Ctx) error {
	a, b, id, e := scope(c, true)
	if e != nil {
		return out(c, e)
	}
	i, e := h.service.Get(c.Context(), a, b, id)
	if e != nil {
		return out(c, e)
	}
	c.Set("ETag", `W/"`+strconv.FormatInt(i.Version, 10)+`"`)
	return c.JSON(i)
}
func (h *Handler) Create(c fiber.Ctx) error {
	if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc.")
	}
	a, b, _, e := scope(c, false)
	if e != nil {
		return out(c, e)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	i, e := h.service.Create(c.Context(), a, b, actor.UserID, input)
	if e != nil {
		return out(c, e)
	}
	return c.Status(201).JSON(i)
}
func (h *Handler) Update(c fiber.Ctx) error {
	a, b, id, e := scope(c, true)
	if e != nil {
		return out(c, e)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	i, e := h.service.Update(c.Context(), a, b, id, actor.UserID, input)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(i)
}
func (h *Handler) SetStatus(c fiber.Ctx) error {
	a, b, id, e := scope(c, true)
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
	i, e := h.service.SetStatus(c.Context(), a, b, id, input.Status, input.Version, actor.UserID)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(i)
}
func out(c fiber.Ctx, e error) error {
	switch {
	case errors.Is(e, ErrInvalid):
		return problem.Write(c, 422, "validation", "Dữ liệu sản phẩm không hợp lệ", "Kiểm tra dữ liệu chung và schema ngành hàng.")
	case errors.Is(e, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy sản phẩm", "Sản phẩm không tồn tại trong workspace.")
	case errors.Is(e, ErrConflict):
		return problem.Write(c, 409, "version-conflict", "Sản phẩm đã thay đổi", "Tải lại sản phẩm trước khi lưu.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý sản phẩm", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
