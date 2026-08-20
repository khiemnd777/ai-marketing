package workspaces

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
func scope(c fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
	clientID, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalid
	}
	workspaceID := uuid.Nil
	if raw := c.Params("workspaceId"); raw != "" {
		workspaceID, err = uuid.Parse(raw)
		if err != nil {
			return uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return clientID, workspaceID, nil
}
func (h *Handler) List(c fiber.Ctx) error {
	clientID, _, err := scope(c)
	if err != nil {
		return respond(c, err)
	}
	items, err := h.service.List(c.Context(), clientID)
	if err != nil {
		return respond(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) Get(c fiber.Ctx) error {
	clientID, id, err := scope(c)
	if err != nil {
		return respond(c, err)
	}
	item, err := h.service.Get(c.Context(), clientID, id)
	if err != nil {
		return respond(c, err)
	}
	c.Set("ETag", `W/"`+strconv.FormatInt(item.Version, 10)+`"`)
	return c.JSON(item)
}
func (h *Handler) Create(c fiber.Ctx) error {
	if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc.")
	}
	clientID, _, err := scope(c)
	if err != nil {
		return respond(c, err)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return respond(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	item, err := h.service.Create(c.Context(), clientID, input, actor, auth.ClientMetadata{RequestID: requestID(c), UserAgent: c.Get(fiber.HeaderUserAgent)})
	if err != nil {
		return respond(c, err)
	}
	return c.Status(201).JSON(item)
}
func (h *Handler) Update(c fiber.Ctx) error {
	clientID, id, err := scope(c)
	if err != nil {
		return respond(c, err)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return respond(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	item, err := h.service.Update(c.Context(), clientID, id, input, actor)
	if err != nil {
		return respond(c, err)
	}
	return c.JSON(item)
}
func (h *Handler) SetStatus(c fiber.Ctx) error {
	clientID, id, err := scope(c)
	if err != nil {
		return respond(c, err)
	}
	var input struct {
		Status  string `json:"status"`
		Version int64  `json:"version"`
	}
	if c.Bind().Body(&input) != nil {
		return respond(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	item, err := h.service.SetStatus(c.Context(), clientID, id, input.Status, input.Version, actor)
	if err != nil {
		return respond(c, err)
	}
	return c.JSON(item)
}
func requestID(c fiber.Ctx) string { v, _ := c.Locals("request_id").(string); return v }
func respond(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalid):
		return problem.Write(c, 422, "validation", "Dữ liệu không hợp lệ", "Vui lòng kiểm tra workspace.")
	case errors.Is(err, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy workspace", "Workspace không tồn tại trong khách hàng này.")
	case errors.Is(err, ErrConflict):
		return problem.Write(c, 409, "conflict", "Workspace đã tồn tại hoặc thay đổi", "Tải lại dữ liệu và thử lại.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý workspace", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
